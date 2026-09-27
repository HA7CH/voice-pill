# Voice Pill

**按住 Fn，说完松手，文字就落在光标所在的位置。**

一个轻量的 macOS 原生语音输入工具。Liquid Glass 胶囊、实时字幕、自动粘贴；没有菜单栏图标，也没有 Dock 常驻图标。

[**下载安装包**](https://github.com/HA7CH/voice-pill/releases/latest) · [English guide](docs/README.en.md) · [隐私说明](docs/PRIVACY.md) · [开源许可](LICENSE)

## 快速安装

**Apple Silicon（M1 或更新）· macOS 26+ · 需要联网。** 当前版本在 macOS 27 上完成构建与启动检查；macOS 26 的实际录音仍欢迎反馈。暂不提供 Intel 安装包。

1. 在 [Releases](https://github.com/HA7CH/voice-pill/releases/latest) 下载 `Voice-Pill-0.1.0-arm64.dmg`。
2. 打开 DMG，将 **Voice Pill** 拖到 **Applications / 应用程序**，弹出安装盘，再从应用程序打开。
3. 首次开放 **麦克风** 和 **辅助功能** 权限，并在应用内确认音频上传到所选转写服务。辅助功能授权后退出并重新打开一次。
4. 在系统设置 → 键盘，将“按下 🌐 键时”改成“不执行操作”。
5. 把光标放进输入框，**按住 Fn 说话，松开即可粘贴**。

豆包模式已自带转写程序和音频库，**不用安装 Homebrew、FFmpeg 或 Codex，也不用填写付费 API Key**。

这是尚未经过 Apple 公证的社区预览版。如果系统拦截，请确认来自本仓库，再到系统设置 → 隐私与安全 → “仍要打开”。不要关闭全局 Gatekeeper。Release 附有 `SHA256SUMS` 用于核对下载文件。由于采用 ad-hoc 签名，更新后可能需要重新添加辅助功能权限。

## 怎么用

| 操作 | 效果 |
| --- | --- |
| 按住 Fn | 豆包实时转写，胶囊显示滚动字幕 |
| 松开 Fn | 将最终文字粘贴到开始录音的应用；不会按回车、发送消息 |
| Ctrl + Fn | 可选 Codex 转写，录音时显示波形 |
| Esc | 取消当前录音 |
| Ctrl + Option + Space | 开始 / 结束录音，服务可在设置里选择 |
| 重新打开 Voice Pill | 打开设置、最新文字、Saved recordings / Retry 和退出按钮 |

轻点 Fn 不触发录音；长按阈值 180 ms。自动标点默认关闭，可在设置中开启。运行时没有菜单栏或 Dock 图标。

### 断线和 Retry

实时连接中断时会继续在本地保存录音，松开 Fn 后尝试用完整录音补转写。失败录音会跨退出、重启保留，打开设置里的 **Saved recordings → Retry** 可重试。网络断开期间字幕可能停止更新。

豆包没有应用侧固定录音时长限制，但第三方服务仍可能中断或限流。Codex 单次最多五分钟。转写成功后删除当前录音；粘贴失败时最新文字保留在设置窗口供复制，退出后不保留文字历史。

### 可选：Codex

另行安装 [codex-asr](https://github.com/Wangnov/codex-asr)，并按其文档完成本机 Codex 登录。Voice Pill 会从 `~/.cargo/bin`、`~/.local/bin`、`/opt/homebrew/bin` 或 `/usr/local/bin` 查找它。之后按 **Ctrl + Fn** 使用。安装包不含 Codex 登录信息，也不包含这个可选程序。

## 数据和服务

- 语音由所选第三方服务处理，**不是离线识别**。豆包采用 FreeASR 的非官方输入法协议，不是官方商业 API；上游将这条集成定位为学习研究使用，服务可能改变或失效。
- 首次上传前需要在应用内同意。HA7CH 不运营语音中转服务器；没有应用内统计上报。
- 失败录音和服务凭证只保存在本机 `~/Library/Application Support/VoicePill/`，不会随源码或安装包发布。
- 玻璃 UI 采用 LiquidType 的实现，使用部分私有渲染参数，不同 macOS 版本可能有视觉差异。可从终端使用 `open -a 'Voice Pill' --args --public-glass` 测试公开玻璃路径（先退出应用）。

完整说明见 [PRIVACY.md](docs/PRIVACY.md)。

## 从源码构建

需要 macOS 26+、Apple Silicon、支持 macOS 26+ SDK 的 Xcode / Command Line Tools、Python 3、Go 1.25+、Homebrew。

```sh
git clone https://github.com/HA7CH/voice-pill.git
cd voice-pill
brew install go opus pkg-config
bash scripts/test.sh
bash scripts/build.sh
open 'build/Voice Pill.app'
```

`build.sh` 编译 Swift、构建修改版 FreeASR、收集动态库和许可证，然后签名。默认是 ad-hoc 开发签名；也可先运行 `python3 scripts/setup-signing.py` 创建本机固定开发身份。已存在的 `.signing/` 不要删除、不要上传，变更身份可能使系统权限失效。

```sh
bash scripts/package.sh  # 生成 dist/ 中的 DMG、ZIP 和校验文件
```

调试预览不录音、不上传：`open 'build/Voice Pill.app' --args --preview-pill --preview-recording --preview-expansion`。退出预览后重新正常启动。界面和构建检查不代表所有输入框、麦克风、网络下都已验证。

## 致谢与开源

- [LiquidType](https://github.com/LuliYanng/LiquidType)：玻璃、胶囊布局、字幕滚动与渐隐、波形和转写动画（MIT）。
- [FreeASR](https://github.com/WEIFENG2333/FreeASR)：豆包输入法协议和 Opus 流式转写；本仓库包含修改后的源码。
- [Whisper](https://github.com/simicvm/whisper)：粘贴实现参考（MIT）。
- [codex-asr](https://github.com/Wangnov/codex-asr)：可选 Codex 后端。

原始代码采用 **MIT**，第三方代码保留各自许可，详见 [THIRD_PARTY.md](THIRD_PARTY.md)。不包含此前试验的微信闭源转写程序。欢迎通过 Issues 提交 macOS 版本、目标 App 和复现步骤；不要上传私人录音或凭证。
