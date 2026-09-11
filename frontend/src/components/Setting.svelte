<script>
import { createEventDispatcher, onMount } from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { GetConfig, UpdateConfig, RegisterGlobalHotkey, SetOpacity } from "../../wailsjs/go/main/App"
    import { ToggleAutoStart, IsAutoStartCheck } from "../../wailsjs/go/internal/AppService"
    import { LogInfo } from '../../wailsjs/runtime/runtime';
    import { internal } from "../../wailsjs/go/models"

    let config = null; // 初始设为 null
    const modifiers = ["Alt", "Ctrl", "Shift", "Win"];
    const keys = ["Space", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "Return", "Escape", "Delete", "Tab", "Left", "Right", "Up", "Down", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"];
    let selectedMod = "";
    let selectedKey = "";

    // 3. 在数据加载完成后，解析 config.shortcuts.wakeUp
    // 比如把 "Alt+Space" 拆分成 "Alt" 和 "Space"
    $: if (config && config.shortcuts.wakeUp && !selectedMod) {
        selectedMod = config.shortcuts.wakeUp[0];
        selectedKey = config.shortcuts.wakeUp[1];
    }

    function updateHotkey() {
        config.shortcuts.wakeUp = [selectedMod,selectedKey];
        LogInfo("新快捷键:" + config.shortcuts.wakeUp);
        RegisterGlobalHotkey(config.shortcuts.wakeUp[0], config.shortcuts.wakeUp[1]);
        UpdateConfig(config);
    }

    function updateOpacity() {
        config.appearance.opacity = Number(config.appearance.opacity);
        LogInfo("新透明度:" + config.appearance.opacity);
        SetOpacity(config.appearance.opacity);
        UpdateConfig(config);
    }

    function updatePasteWaitTime() {
        config.shortcuts.pasteWaitTime = Number(config.shortcuts.pasteWaitTime);
        LogInfo("新等待时间:" + config.shortcuts.pasteWaitTime);
        UpdateConfig(config);
    }    

    onMount(async () => {
        try {
            const rawConfig = await GetConfig();
            config = internal.Config.createFrom(rawConfig);
        } catch (error) {
            console.error('Failed to load config:', error);
        }

        try {
            // 1. 页面加载时从系统读取真实的自启状态
            const isAutoStart = await IsAutoStartCheck();
            if (isAutoStart != config.general.launchAtLogin) {
                config.general.launchAtLogin = isAutoStart;
                save();
            }
        } catch (err) {
            console.error("读取自启状态失败:", err);
        }
    });

    const dispatch = createEventDispatcher();

    const tabs = [
        { id: 'general', label: '常规 (General)', icon: '⚙️' },
        { id: 'shortcuts', label: '快捷键 (Hotkeys)', icon: '⌨️' },
        { id: 'appearance', label: '外观 (Appearance)', icon: '🎨' },
        { id: 'about', label: '关于 (About)', icon: 'ℹ️' },
    ];

    let activeTab = 'general';

    function close() {
        dispatch('close');
    }

    // 保存设置
    function save() {
        // TODO: 调用 Wails SaveConfig(config)
        UpdateConfig(config);
        close();
    }

    async function syncAutoStart(enabled) {
        try {
            await ToggleAutoStart(enabled);
            LogInfo("开机自启状态已更新:" + enabled);
        } catch (err) {
            LogInfo("设置自启失败:" + err);
            // 如果失败，可以考虑把前端状态回滚
            // config.general.launchAtLogin = !enabled;
        }
    }

</script>

<!-- 遮罩层：点击空白处关闭 -->
 {#if config}
    <div class="overlay" transition:fade={{duration: 100}} on:click={close} on:keydown={e => e.key === 'Escape' && close()}>
        
        <!-- 设置窗口主体 -->
        <div class="settings-window" transition:fly={{y: 10, duration: 200}} on:click|stopPropagation on:keydown={e => e.key === 'Escape' && close()}>
            
            <!-- 左侧侧边栏 -->
            <div class="sidebar">
                <div class="sidebar-title">Settings</div>
                <ul class="nav-list">
                    {#each tabs as tab}
                        <li
                            class:active={activeTab === tab.id} 
                            on:click={() => activeTab = tab.id}
                            on:keydown={e => e.key === 'Escape' && close()}>
                            <span class="nav-icon">{tab.icon}</span>
                            {tab.label}
                        </li>
                    {/each}
                </ul>
            </div>

            <!-- 右侧内容区 -->
            <div class="content">
                <div class="content-header">
                    <h2>{tabs.find(t => t.id === activeTab)?.label || ''}</h2>
                </div>

                <div class="content-body">
                    <!-- Tab 1: 常规设置 -->
                    {#if activeTab === 'general'}
                        <div class="setting-group" in:fade={{duration:150}}>
                            <div class="setting-row">
                                <div class="setting-info">
                                    <span class="setting-title">开机自启</span>
                                    <span class="desc">登录时自动启动 Quick-Clip</span>
                                </div>
                                <!-- iOS 风格开关 -->
                                <label class="toggle-switch">
                                    <input type="checkbox" 
                                    bind:checked={config.general.launchAtLogin} 
                                    on:change={() => syncAutoStart(config.general.launchAtLogin)}>
                                    <span class="slider"></span>
                                </label>
                            </div>
                        </div>
                    {/if}

                    <!-- Tab 2: 快捷键 -->
                    {#if activeTab === 'shortcuts'}
                        <div class="setting-group" in:fade={{duration:150}}>
                            <div class="setting-row">
                                <div class="setting-info">
                                    <span class="setting-title">唤醒快捷键</span>
                                    <span class="desc">组合键唤醒主窗口</span>
                                </div>
                                
                                <div class="hotkey-picker">
                                    <!-- 修饰键下拉框 -->
                                    <select class="styled-select" bind:value={selectedMod} on:change={updateHotkey}>
                                        {#each modifiers as mod}
                                            <option value={mod}>{mod}</option>
                                        {/each}
                                    </select>

                                    <span class="plus-sign">+</span>

                                    <!-- 主键下拉框 -->
                                    <select class="styled-select" bind:value={selectedKey} on:change={updateHotkey}>
                                        {#each keys as k}
                                            <option value={k}>{k}</option>
                                        {/each}
                                    </select>
                                </div>
                            </div>

                            <div class="setting-row">
                                <div class="setting-info">
                                    <span class="setting-title">粘贴等待时间</span>
                                    <span class="desc">粘贴操作前等待的时间,应对卡顿,默认100ms</span>
                                </div>

                                <div class="range-wrapper">
                                    <input type="range" min="25" max="1000" step="25" 
                                    bind:value={config.shortcuts.pasteWaitTime}
                                    on:change={updatePasteWaitTime}
                                    >
                                    <span>{config.shortcuts.pasteWaitTime}ms</span>
                                </div>
                            </div>
                        </div>
                    {/if}

                    <!-- Tab 3: 外观 -->
                    {#if activeTab === 'appearance'}
                        <div class="setting-group" in:fade={{duration:150}}>
                            <div class="setting-row">
                                <div class="setting-info">
                                    <span class="setting-title">窗口透明度</span>
                                </div>
                                <div class="range-wrapper">
                                    <input type="range" min="25" max="255" step="1" 
                                    bind:value={config.appearance.opacity}
                                    on:change={updateOpacity}
                                    >
                                    <span>{(config.appearance.opacity / 255).toFixed(2)}%</span>
                                </div>
                            </div>
                        </div>
                    {/if}

                    <!-- Tab 4: 关于 -->
                    {#if activeTab === 'about'}
                    <div class="about-section" in:fade={{duration:150}}>
                        <h3>Quick-Clip</h3>
                        <p>@Drawye</p>
                        <p class="desc">A compact clipboard manager for efficiency.</p>
                    </div>
                {/if}
                </div>

                <!-- 底部按钮 -->
                <div class="content-footer">
                    <button class="btn-cancel" on:click={close}>返回</button>
                    <!-- <button class="btn-save" on:click={save}>保存修改</button> -->
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    /* 全局变量继承你的 App 风格 */
    * { box-sizing: border-box; }

    .overlay {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(202, 214, 224, 0.24);
        -webkit-backdrop-filter: blur(10px) saturate(1.15);
        backdrop-filter: blur(10px) saturate(1.15);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
    }

    .settings-window {
        width: 500px;
        height: 350px;
        background: rgba(248, 250, 252, 0.8);
        -webkit-backdrop-filter: blur(30px) saturate(1.28);
        backdrop-filter: blur(30px) saturate(1.28);
        border-radius: 8px;
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.76),
            0 16px 44px rgba(24, 39, 53, 0.22);
        display: flex;
        overflow: hidden;
        border: 1px solid rgba(255, 255, 255, 0.72);
        font-size: 13px;
    }

    /* --- 侧边栏 --- */
    .sidebar {
        width: 140px;
        background: rgba(230, 238, 244, 0.48);
        border-right: 1px solid rgba(72, 99, 121, 0.1);
        box-shadow: inset 1px 0 0 rgba(255, 255, 255, 0.46);
        display: flex;
        flex-direction: column;
        padding: 10px 0;
    }

    .sidebar-title {
        padding: 0 16px 10px;
        font-weight: 600;
        color: #526678;
        font-size: 11px;
        letter-spacing: 0;
        text-transform: uppercase;
    }

    .nav-list {
        list-style: none;
        padding: 0; margin: 0;
    }

    .nav-list li {
        padding: 8px 16px;
        cursor: pointer;
        color: #444;
        transition: all 0.2s;
        display: flex;
        align-items: center;
    }

    .nav-icon { margin-right: 8px; font-size: 14px; opacity: 0.8; }

    .nav-list li:hover { background: rgba(0,0,0,0.05); }
    
    .nav-list li.active {
        background: rgba(219, 237, 247, 0.72);
        color: #234b63;
        font-weight: 500;
        border-left: 2px solid rgba(42, 137, 188, 0.78);
        box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.42);
    }

    /* --- 内容区 --- */
    .content {
        flex: 1;
        display: flex;
        flex-direction: column;
        background: rgba(255, 255, 255, 0.22);
    }

    .content-header {
        padding: 12px 20px;
        border-bottom: 1px solid rgba(75, 99, 119, 0.09);
        box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.42);
    }
    .content-header h2 {
        margin: 0;
        color: #293b4a;
        font-size: 15px;
        font-weight: 600;
        letter-spacing: 0;
    }

    .content-body {
        flex: 1;
        padding: 20px;
        overflow-y: auto;
    }

    /* 通用设置行样式 */
    .setting-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
    }

    .setting-info {
        display: flex;
        flex-direction: column;
    }

    .setting-info .setting-title { font-weight: 500; color: #2e3d4a; margin-bottom: 2px; }
    .setting-info .desc { color: #71808e; font-size: 10px; }

    .content-footer {
        padding: 10px 20px;
        border-top: 1px solid rgba(75, 99, 119, 0.09);
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        background: rgba(239, 245, 249, 0.38);
    }

    button { font-size: 13px; padding: 6px 14px; border-radius: 4px; cursor: pointer; border: 1px solid transparent; }
    .btn-cancel { background: transparent; color: #666; }
    .btn-cancel:hover { color: #333; background: #e0e0e0; }

    /* iOS 风格 Toggle Switch */
    .toggle-switch { position: relative; width: 36px; height: 20px; display: inline-block; }
    .toggle-switch input { opacity: 0; width: 0; height: 0; }
    .slider {
        position: absolute; cursor: pointer;
        top: 0; left: 0; right: 0; bottom: 0;
        background-color: #ccc; transition: .3s; border-radius: 20px;
    }
    .slider:before {
        position: absolute; content: "";
        height: 16px; width: 16px; left: 2px; bottom: 2px;
        background-color: white; transition: .3s; border-radius: 50%;
    }
    input:checked + .slider { background-color: #3b82f6; }
    input:checked + .slider:before { transform: translateX(16px); }

    /* 滑动条 */
    .range-wrapper { display: flex; align-items: center; gap: 10px; }

    .about-section { text-align: center; margin-top: 40px; }
    .about-section h3 { margin: 0 0 10px 0; }
    .about-section .desc { color: #888; }
.hotkey-picker {
        display: flex;
        align-items: center;
        background: #f1f1f4; /* 浅灰色底座 */
        padding: 4px;
        border-radius: 8px;
        gap: 4px;
        border: 1px solid rgba(0,0,0,0.05);
    }

    .styled-select {
        appearance: none; /* 隐藏原生箭头 */
        background: #ffffff;
        border: 1px solid rgba(0,0,0,0.1);
        border-radius: 6px;
        padding: 5px 12px;
        font-size: 12px;
        font-weight: 500;
        color: #333;
        outline: none;
        cursor: pointer;
        min-width: 80px;
        text-align: center;
        box-shadow: 0 1px 2px rgba(0,0,0,0.05);
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    }

    /* 悬停效果 */
    .styled-select:hover {
        background: #fafafa;
        border-color: rgba(0,0,0,0.2);
        transform: translateY(-1px);
        box-shadow: 0 3px 6px rgba(0,0,0,0.08);
    }

    /* 聚焦状态 */
    .styled-select:focus {
        border-color: #3b82f6;
        box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
    }

    .plus-sign {
        font-size: 14px;
        font-weight: 600;
        color: #a1a1aa;
        padding: 0 4px;
        user-select: none;
    }
</style>