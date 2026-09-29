//! Experimental Codex Desktop dictation protocol. Credentials stay in headers.
//! Input is raw mono signed PCM16 little endian; events are complete text snapshots.
use std::{io::Read, time::Duration};

use base64::prelude::*;
use futures_util::{SinkExt, StreamExt};
use serde_json::{json, Value};
use tokio::{
    io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt},
    net::TcpStream,
    sync::mpsc,
    time::{timeout, Instant},
};
use tokio_tungstenite::{
    client_async_tls_with_config,
    tungstenite::{client::IntoClientRequest, Message},
    WebSocketStream,
};

use crate::CodexAuth;

pub type StreamResult<T> = Result<T, Box<dyn std::error::Error + Send + Sync>>;
pub const STREAM_ENDPOINT: &str =
    "wss://chatgpt.com/backend-api/dictation/stream?dictation_surface=composer";

#[derive(Clone, Debug)]
pub struct StreamOptions {
    pub sample_rate: u32,
    pub language: Option<String>,
    pub proxy: Option<String>,
    pub connect_timeout: Duration,
    pub finish_timeout: Duration,
    pub session_timeout: Duration,
}

impl Default for StreamOptions {
    fn default() -> Self {
        Self {
            sample_rate: 24_000,
            language: None,
            proxy: None,
            connect_timeout: Duration::from_secs(15),
            finish_timeout: Duration::from_secs(60),
            session_timeout: Duration::from_secs(300),
        }
    }
}

impl StreamOptions {
    pub fn validate(&self) -> StreamResult<()> {
        if !(8_000..=48_000).contains(&self.sample_rate) {
            return Err("sample rate must be between 8000 and 48000 Hz".into());
        }
        if self.connect_timeout.is_zero()
            || self.finish_timeout.is_zero()
            || self.session_timeout.is_zero()
        {
            return Err("stream timeouts must be positive".into());
        }
        Ok(())
    }

    fn start_message(&self) -> Value {
        let mut value = json!({"type":"session.start","config":{
            "input_audio_format":"pcm16","sample_rate_hz":self.sample_rate,"num_channels":1,
            "max_buffer_size_bytes":4194304,"max_utterance_duration_ms":30000,"session_ttl_ms":300000,
            "provider_mode":"streaming_sse","transcript_delivery_mode":"segment",
            "vad":{"type":"server_vad","threshold":0.5,"prefix_padding_ms":300,"silence_duration_ms":500}
        }});
        if let Some(language) = &self.language {
            value["config"]["language"] = json!(language);
        }
        value
    }
}

#[derive(Default)]
struct Utterance {
    id: String,
    partial: Option<(u64, String)>,
    final_text: Option<(u64, String)>,
}

#[derive(Default)]
struct Transcript {
    utterances: Vec<Utterance>,
}

impl Transcript {
    fn apply(&mut self, event: &Value) -> StreamResult<bool> {
        let kind = event["type"].as_str().unwrap_or_default();
        if !matches!(
            kind,
            "speech.started" | "transcript.segment" | "transcript.final"
        ) {
            return Ok(false);
        }
        let id = event["utterance_id"]
            .as_str()
            .ok_or("missing utterance id")?;
        let index = match self.utterances.iter().position(|u| u.id == id) {
            Some(index) => index,
            None => {
                self.utterances.push(Utterance {
                    id: id.to_owned(),
                    ..Default::default()
                });
                self.utterances.len() - 1
            }
        };
        if kind == "speech.started" {
            return Ok(false);
        }
        let revision = event["revision"]
            .as_u64()
            .ok_or("missing transcript revision")?;
        let text = event["text"].as_str().ok_or("missing transcript text")?;
        let entry = &mut self.utterances[index];
        if kind == "transcript.final" {
            if entry
                .final_text
                .as_ref()
                .is_some_and(|(r, _)| revision < *r)
            {
                return Ok(false);
            }
            entry.final_text = Some((revision, text.to_owned()));
            entry.partial = None;
        } else {
            if entry.final_text.is_some()
                || entry.partial.as_ref().is_some_and(|(r, _)| revision < *r)
            {
                return Ok(false);
            }
            entry.partial = Some((revision, text.to_owned()));
        }
        Ok(true)
    }

    fn text(&self) -> String {
        self.utterances
            .iter()
            .filter_map(|u| u.final_text.as_ref().or(u.partial.as_ref()))
            .map(|(_, text)| text.trim())
            .filter(|s| !s.is_empty())
            .collect::<Vec<_>>()
            .join(" ")
    }

    fn result(&self) -> StreamResult<String> {
        if self.utterances.iter().any(|u| u.final_text.is_none()) {
            return Err(
                "stream closed with unfinished utterances; retain the audio for retry".into(),
            );
        }
        Ok(self.text())
    }
}

// Read on a dedicated thread: Tokio's blocking stdin otherwise prevents runtime
// shutdown after a network failure while the caller keeps its pipe open.
fn audio_reader(
    mut reader: Box<dyn Read + Send>,
    chunk_bytes: usize,
) -> mpsc::Receiver<std::io::Result<Vec<u8>>> {
    let (tx, rx) = mpsc::channel(8);
    std::thread::spawn(move || {
        let mut odd = None;
        loop {
            let mut bytes = vec![0; chunk_bytes];
            let offset = usize::from(odd.is_some());
            if let Some(byte) = odd.take() {
                bytes[0] = byte;
            }
            let size = match reader.read(&mut bytes[offset..]) {
                Ok(0) => {
                    if offset != 0 {
                        let _ = tx.blocking_send(Err(std::io::Error::new(
                            std::io::ErrorKind::InvalidData,
                            "truncated PCM16 sample",
                        )));
                    }
                    return;
                }
                Ok(n) => n + offset,
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {
                    if offset != 0 {
                        odd = Some(bytes[0]);
                    }
                    continue;
                }
                Err(e) => {
                    let _ = tx.blocking_send(Err(e));
                    return;
                }
            };
            bytes.truncate(size);
            if bytes.len() % 2 != 0 {
                odd = bytes.pop();
            }
            if !bytes.is_empty() && tx.blocking_send(Ok(bytes)).is_err() {
                return;
            }
        }
    });
    rx
}

async fn transport(proxy: Option<&str>) -> StreamResult<TcpStream> {
    let Some(proxy) = proxy else {
        return Ok(TcpStream::connect(("chatgpt.com", 443)).await?);
    };
    let url = reqwest::Url::parse(proxy).map_err(|_| "invalid stream proxy URL")?;
    // Do not silently bypass a configured proxy. TLS still terminates at ChatGPT.
    if url.scheme() != "http" {
        return Err("stream supports HTTP CONNECT proxies; set --proxy http://host:port".into());
    }
    let host = url.host_str().ok_or("proxy host is missing")?;
    let port = url.port_or_known_default().ok_or("proxy port is missing")?;
    let mut stream = TcpStream::connect((host, port)).await?;
    let mut connect = "CONNECT chatgpt.com:443 HTTP/1.1\r\nHost: chatgpt.com:443\r\n".to_owned();
    if !url.username().is_empty() {
        let auth = BASE64_STANDARD.encode(format!(
            "{}:{}",
            url.username(),
            url.password().unwrap_or_default()
        ));
        connect.push_str(&format!("Proxy-Authorization: Basic {auth}\r\n"));
    }
    connect.push_str("\r\n");
    stream.write_all(connect.as_bytes()).await?;
    let mut response = Vec::new();
    while !response.ends_with(b"\r\n\r\n") {
        if response.len() >= 16_384 {
            return Err("oversized proxy response".into());
        }
        response.push(stream.read_u8().await?);
    }
    let response = String::from_utf8_lossy(&response);
    if response
        .lines()
        .next()
        .and_then(|s| s.split_whitespace().nth(1))
        != Some("200")
    {
        return Err("HTTP proxy rejected CONNECT".into());
    }
    Ok(stream)
}

/// Emits `ready`, `partial`, `final`, `input.finished`, and a single `result`.
/// Only `result` means all utterances finished and the server acknowledged close.
/// On failure the caller must retain captured audio; this function never retries
/// live input automatically, since replaying incomplete audio can lose words.
pub async fn transcribe<F>(
    auth: &CodexAuth,
    options: StreamOptions,
    reader: Box<dyn Read + Send>,
    pace: bool,
    emit: F,
) -> StreamResult<()>
where
    F: FnMut(Value) -> StreamResult<()>,
{
    options.validate()?;
    let connect = async {
        let mut request = STREAM_ENDPOINT.into_client_request()?;
        let headers = request.headers_mut();
        let mut bearer = format!("Bearer {}", auth.access_token)
            .parse::<tokio_tungstenite::tungstenite::http::HeaderValue>()
            .map_err(|_| "invalid authorization header")?;
        bearer.set_sensitive(true);
        headers.insert("authorization", bearer);
        headers.insert("originator", "Codex Desktop".parse()?);
        headers.insert("user-agent", "Codex Desktop/26.924.22138".parse()?);
        headers.insert(
            "sec-websocket-protocol",
            "chatgpt-dictation, codex-desktop".parse()?,
        );
        if let Some(id) = &auth.account_id {
            headers.insert(
                "chatgpt-account-id",
                id.parse().map_err(|_| "invalid account header")?,
            );
        }
        let transport = transport(options.proxy.as_deref()).await?;
        let (ws, _) = client_async_tls_with_config(request, transport, None, None)
            .await
            .map_err(|error| {
                // HTTP response bodies/headers can contain sensitive routing data.
                match error {
                    tokio_tungstenite::tungstenite::Error::Http(response) => format!(
                        "dictation handshake rejected: HTTP {}",
                        response.status().as_u16()
                    ),
                    _ => "dictation connection failed (network, TLS or WebSocket handshake)"
                        .to_owned(),
                }
            })?;
        Ok::<_, Box<dyn std::error::Error + Send + Sync>>(ws)
    };
    let ws = timeout(options.connect_timeout, connect)
        .await
        .map_err(|_| "dictation connection timed out")??;
    let rx = audio_reader(reader, (options.sample_rate as usize / 10) * 2);
    tokio::select! {
        result = timeout(options.connect_timeout + options.session_timeout + options.finish_timeout,
            session(ws, &options, rx, pace, emit)) => {
            result.map_err(|_| "dictation transport stalled; retain audio for retry")?
        },
        _ = tokio::signal::ctrl_c() => Err("dictation cancelled".into()),
    }
}

async fn session<S, F>(
    mut ws: WebSocketStream<S>,
    options: &StreamOptions,
    mut audio: mpsc::Receiver<std::io::Result<Vec<u8>>>,
    pace: bool,
    mut emit: F,
) -> StreamResult<()>
where
    S: AsyncRead + AsyncWrite + Unpin,
    F: FnMut(Value) -> StreamResult<()>,
{
    let start = Instant::now();
    let mut started = false;
    let mut finishing = false;
    let mut transcript = Transcript::default();
    let mut deadline = start + options.connect_timeout;
    let mut next_audio = start;
    let mut bytes_sent = 0_u64;
    let mut send_event = |mut event: Value| {
        event["elapsed_ms"] = json!(start.elapsed().as_millis() as u64);
        emit(event)
    };
    ws.send(Message::Text(options.start_message().to_string().into()))
        .await?;
    loop {
        tokio::select! {
            _ = tokio::time::sleep_until(deadline) => return Err(if finishing { "timed out waiting for final transcript; retain audio for retry" } else { "dictation session timed out; retain audio for retry" }.into()),
            incoming = ws.next() => {
                match incoming {
                    Some(Ok(Message::Text(text))) => {
                        let event: Value = serde_json::from_str(&text).map_err(|_| "invalid dictation event")?;
                        match event["type"].as_str().unwrap_or_default() {
                            "session.started" => {
                                if started { return Err("duplicate session.started".into()); }
                                started = true;
                                deadline = Instant::now() + options.session_timeout;
                                next_audio = Instant::now();
                                send_event(json!({"type":"ready","sample_rate":options.sample_rate}))?;
                            }
                            "speech.started" | "transcript.segment" | "transcript.final" => {
                                if transcript.apply(&event)? {
                                    send_event(json!({"type":if event["type"] == "transcript.final" { "final" } else { "partial" }, "text":transcript.text(), "utterance_id":event["utterance_id"], "revision":event["revision"]}))?;
                                }
                            }
                            "transcript.failed" | "session.error" => return Err("upstream dictation failed; retain audio for retry".into()),
                            "session.updated" if event["session"]["status"] == "closed" => {
                                if !finishing { return Err("server closed before all input was sent; retain audio for retry".into()); }
                                let text = transcript.result()?;
                                send_event(json!({"type":"result","text":text}))?;
                                let _ = timeout(Duration::from_secs(1), ws.close(None)).await;
                                return Ok(());
                            }
                            _ => {},
                        }
                    }
                    Some(Ok(Message::Ping(payload))) => { ws.send(Message::Pong(payload)).await?; }
                    Some(Ok(Message::Pong(_))) => {},
                    Some(Ok(Message::Close(_))) | None => return Err("dictation disconnected before completion; retain audio for retry".into()),
                    Some(Err(_)) => return Err("dictation transport failed; retain audio for retry".into()),
                    _ => return Err("unexpected binary dictation event".into()),
                }
            }
            chunk = async {
                if pace { tokio::time::sleep_until(next_audio).await; }
                audio.recv().await
            }, if started && !finishing => {
                match chunk {
                    Some(chunk) => {
                        let chunk = chunk?;
                        bytes_sent += chunk.len() as u64;
                        ws.send(Message::Text(json!({"type":"audio.append","audio":BASE64_STANDARD.encode(&chunk)}).to_string().into())).await?;
                        next_audio = Instant::now() + Duration::from_secs_f64(chunk.len() as f64 / (options.sample_rate as f64 * 2.0));
                    }
                    None => {
                        finishing = true;
                        deadline = Instant::now() + options.finish_timeout;
                        send_event(json!({"type":"input.finished","audio_bytes":bytes_sent}))?;
                        ws.send(Message::Text(json!({"type":"session.close"}).to_string().into())).await?;
                    }
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn revisions_replace_instead_of_appending_and_final_is_authoritative() {
        let mut t = Transcript::default();
        for event in [
            json!({"type":"transcript.segment","utterance_id":"a","revision":2,"text":"hello world"}),
            json!({"type":"transcript.segment","utterance_id":"a","revision":1,"text":"wrong"}),
            json!({"type":"transcript.final","utterance_id":"a","revision":3,"text":"Hello world."}),
            json!({"type":"transcript.segment","utterance_id":"a","revision":4,"text":"wrong again"}),
        ] {
            t.apply(&event).unwrap();
        }
        assert_eq!(t.result().unwrap(), "Hello world.");
    }

    #[test]
    fn speech_order_survives_out_of_order_final_results() {
        let mut t = Transcript::default();
        for id in ["a", "b"] {
            t.apply(&json!({"type":"speech.started","utterance_id":id}))
                .unwrap();
        }
        t.apply(
            &json!({"type":"transcript.final","utterance_id":"b","revision":1,"text":"second"}),
        )
        .unwrap();
        assert!(t.result().is_err());
        t.apply(&json!({"type":"transcript.final","utterance_id":"a","revision":1,"text":"first"}))
            .unwrap();
        assert_eq!(t.result().unwrap(), "first second");
    }

    #[tokio::test]
    async fn pcm_reader_rejects_a_truncated_sample() {
        let mut rx = audio_reader(Box::new(std::io::Cursor::new(vec![0, 1, 2])), 4);
        assert_eq!(rx.recv().await.unwrap().unwrap(), vec![0, 1]);
        assert!(rx.recv().await.unwrap().is_err());
    }

    async fn mock_session(finalize: bool) -> StreamResult<Vec<Value>> {
        let (client, server) = tokio::io::duplex(16384);
        let server_task = tokio::spawn(async move {
            let mut ws = tokio_tungstenite::accept_async(server).await.unwrap();
            ws.next().await.unwrap().unwrap();
            ws.send(Message::Text(
                json!({"type":"session.started"}).to_string().into(),
            ))
            .await
            .unwrap();
            loop {
                let Some(Ok(Message::Text(message))) = ws.next().await else {
                    break;
                };
                let event: Value = serde_json::from_str(&message).unwrap();
                if event["type"] == "audio.append" {
                    ws.send(Message::Text(json!({"type":"transcript.segment","utterance_id":"a","revision":1,"text":"partial"}).to_string().into())).await.unwrap();
                } else if event["type"] == "session.close" {
                    if finalize {
                        ws.send(Message::Text(json!({"type":"transcript.final","utterance_id":"a","revision":2,"text":"final"}).to_string().into())).await.unwrap();
                    }
                    ws.send(Message::Text(
                        json!({"type":"session.updated","session":{"status":"closed"}})
                            .to_string()
                            .into(),
                    ))
                    .await
                    .unwrap();
                    break;
                }
            }
        });
        let (ws, _) = tokio_tungstenite::client_async("ws://localhost", client).await?;
        let rx = audio_reader(Box::new(std::io::Cursor::new(vec![0; 4800])), 4800);
        let mut output = Vec::new();
        let result = session(ws, &StreamOptions::default(), rx, false, |e| {
            output.push(e);
            Ok(())
        })
        .await;
        server_task.await?;
        result?;
        Ok(output)
    }

    #[tokio::test]
    async fn final_result_requires_close_ack_and_final_transcripts() {
        let output = mock_session(true).await.unwrap();
        assert_eq!(output.last().unwrap()["type"], "result");
        assert_eq!(output.last().unwrap()["text"], "final");
        assert!(output.iter().any(|e| e["type"] == "partial"));
        assert!(mock_session(false).await.is_err());
    }

    #[tokio::test]
    async fn disconnect_does_not_wait_for_stdin_to_close() {
        let (client, server) = tokio::io::duplex(16384);
        let server_task = tokio::spawn(async move {
            let mut ws = tokio_tungstenite::accept_async(server).await.unwrap();
            ws.next().await.unwrap().unwrap();
            ws.send(Message::Text(
                json!({"type":"session.started"}).to_string().into(),
            ))
            .await
            .unwrap();
            ws.close(None).await.unwrap();
        });
        let (ws, _) = tokio_tungstenite::client_async("ws://localhost", client)
            .await
            .unwrap();
        let (_keep_input_open, rx) = mpsc::channel(8);
        let mut output = Vec::new();
        let result = timeout(
            Duration::from_secs(1),
            session(ws, &StreamOptions::default(), rx, false, |e| {
                output.push(e);
                Ok(())
            }),
        )
        .await
        .unwrap();
        assert!(result.is_err());
        assert!(!output.iter().any(|e| e["type"] == "result"));
        server_task.await.unwrap();
    }

    #[tokio::test]
    async fn missing_start_ack_times_out() {
        let (client, server) = tokio::io::duplex(16384);
        let (ws, server_ws) = tokio::join!(
            tokio_tungstenite::client_async("ws://localhost", client),
            tokio_tungstenite::accept_async(server)
        );
        let (ws, _) = ws.unwrap();
        let _server_ws = server_ws.unwrap();
        let (_keep_input_open, rx) = mpsc::channel(8);
        let options = StreamOptions {
            connect_timeout: Duration::from_millis(20),
            ..Default::default()
        };
        let result = timeout(
            Duration::from_secs(1),
            session(ws, &options, rx, false, |_| Ok(())),
        )
        .await
        .unwrap();
        assert!(result.is_err());
    }
}
