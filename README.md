# Quick-Clip

Quick-Clip 是一款基于wails开发的专为效率控设计的轻量级、开源桌面密码管理助手。它不仅能安全地存储你的敏感信息，更能通过“一键穿梭”功能，自动将密码填入目标程序，彻底告别繁琐的复制粘贴。

<img width="320" height="480" alt="image" src="https://github.com/user-attachments/assets/b0ead354-fe1d-42b5-bc22-090815aa93bc" />


✨ 核心特性

🚀 极速唤醒：支持自定义全局快捷键（如 Alt + Space），随叫随到，用完即隐藏。

⌨️ 智能填入：点击条目后，程序会自动切换回上一个活动窗口，并模拟键盘输入密码，支持所有桌面应用与浏览器。

🔐 军事级加密：所有数据均采用 AES-256 算法进行本地加密存储，不上传云端，确保隐私掌握在自己手中。

📂 树状组织：支持多级目录分类，清晰管理不同场景（如测试、社交、金融）的密码。

🪶 极致轻量：基于 Wails & WebView2 构建，安装包极小，运行时内存占用仅 100MB 左右。

🎨 现代 UI：精致的侧边栏设计，支持透明度调节，完美融入你的桌面审美。

🛠️ 工作原理

呼出：按下全局快捷键，Quick-Clip 浮现在屏幕中央。

搜索：通过顶部的搜索框快速定位需要的账户。

填入：点击目标条目，Quick-Clip 会：

自动隐藏当前窗口。

利用系统底层接口回溯至上一个聚焦的应用程序。

模拟键盘敲击，将密码字符逐个填入光标所在位置。

🔒 安全说明

本地存储：数据文件存储在用户本地目录下，不提供后端接口，从物理上杜绝了拖库风险。

加密协议：使用随机 Salt 与 AES 算法对 JSON 数据库进行全盘加密。

开源透明：核心逻辑清晰可见，你可以放心编译使用。

🚀 快速开始

下载运行

⚙️ 设置选项

通过托盘菜单进入 Settings 页面，你可以：

常规：开启/关闭开机自启。

快捷键：配置最顺手的唤醒组合键。

Quick-Clip —— 让密码管理回归简单与高效。


# Quick-Clip

Quick-Clip is a lightweight, open-source desktop password management assistant developed with Wails, designed for efficiency enthusiasts. It not only securely stores your sensitive information but also features "One-Click Navigation" to automatically fill passwords into target applications, eliminating the need for cumbersome copy-paste operations.

✨ **Core Features**

🚀 **Instant Wake-up**: Supports customizable global hotkeys (e.g., Alt + Space), available on demand, and hides immediately after use.

⌨️ **Smart Auto-fill**: After clicking an entry, the application automatically switches back to the last active window and simulates keyboard input for passwords, compatible with all desktop applications and browsers.

🔐 **Military-Grade Encryption**: All data is locally encrypted using AES-256 algorithm, stored offline without cloud uploads, ensuring your privacy remains in your hands.

📂 **Hierarchical Organization**: Supports multi-level directory categorization for clear management of passwords across different scenarios (e.g., testing, social media, finance).

🪶 **Extremely Lightweight**: Built with Wails & WebView2, the installation package is minimal, with runtime memory usage around 100MB.

🎨 **Modern UI**: Elegant sidebar design with adjustable transparency, seamlessly integrating with your desktop aesthetics.

🛠️ **How It Works**

**Invoke**: Press the global hotkey to bring Quick-Clip to the center of your screen.

**Search**: Quickly locate the desired account using the top search bar.

**Fill**: Click on the target entry, and Quick-Clip will:
   - Automatically hide its current window.
   - Utilize system-level APIs to return to the previously focused application.
   - Simulate keyboard strokes to input password characters one by one at the cursor position.

🔒 **Security Notes**

**Local Storage**: Data files are stored in the user's local directory, with no backend interface, physically eliminating the risk of database breaches.

**Encryption Protocol**: Uses random Salt and AES algorithm for full-disk encryption of the JSON database.

**Open Source Transparency**: Core logic is clearly visible, allowing you to compile and use with confidence.

🚀 **Quick Start**

**Download and Run**

⚙️ **Settings Options**

Access the Settings page via the system tray menu, where you can:
   - **General**: Enable/disable auto-start on boot.
   - **Hotkeys**: Configure your preferred wake-up key combinations.

**Quick-Clip** — Simplifying password management with efficiency and ease.
