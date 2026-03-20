# 银河麒麟 TUI 管理面板

一个给银河麒麟 `V10` 用的终端管理面板，偏系统运维和桌面维护。

## 功能

- 合并查看系统信息和网络信息
- 分析磁盘文件占用
- 合并展示 CPU 和内存占用
- 编辑并保存网卡 IPv4 配置
- 切换银河麒麟软件源
- 清理包缓存
- 清理 `.log` 文件
- 安装固定软件清单

## 一行运行

安装并启动：

```bash
curl -fsSL https://raw.githubusercontent.com/chxcodepro/qilin-manager-tui/main/scripts/install.sh | bash
```

直接下载后启动：

```bash
curl -L https://github.com/chxcodepro/qilin-manager-tui/releases/latest/download/qilin-manager-tui_linux_amd64.tar.gz | tar -xz && ./qilin-manager-tui
```

`arm64` 把命令里的 `amd64` 换成 `arm64` 就行。

## 页面说明

- `系统/网络`
- `磁盘分析`
- `CPU/内存`
- `软件维护`

## 软件维护页按键

- `o` 切换到内置官方源
- `b` 恢复备份源
- `u` 更新软件索引
- `c` 清理包缓存
- `g` 清理 `.log` 文件
- `↑/↓` 选择软件
- `空格` 勾选软件
- `i` 安装勾选的软件

## 其他按键

- `系统/网络` 页
  - `↑/↓` 选择网卡
  - `e` 编辑网卡
  - `Ctrl+S` 保存网卡配置
- `磁盘分析` 页
  - `↑/↓` 选择子项
  - `Enter` 进入目录
  - `Backspace` 返回上一级

## 固定软件清单

- `wps-office`
- `electronic-wechat`
- `linuxqq`
- `netdisk`
- `kylin-software-center`
