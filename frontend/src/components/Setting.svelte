<script>
import { createEventDispatcher, onMount } from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { GetConfig, UpdateConfig } from "../../wailsjs/go/main/App"
    import { ToggleAutoStart, IsAutoStartCheck } from "../../wailsjs/go/internal/AppService"
    import { LogInfo } from '../../wailsjs/runtime/runtime';
    import { internal } from "../../wailsjs/go/models"

    let config = null; // 初始设为 null

    onMount(async () => {
        try {
            const rawConfig = await GetConfig();
            // 调试用：看看后端到底传过来了什么
            console.log("Raw config from Go:", rawConfig); 
            config = internal.Config.createFrom(rawConfig);
        } catch (error) {
            console.error('Failed to load config:', error);
        }

        try {
            // 1. 页面加载时从系统读取真实的自启状态
            const isAutoStart = await IsAutoStartCheck();
            if (isAutoStart != config.general.launchAtLogin) {
                config.general.launchAtLogin = isAutoStart;
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

    // 简单的键盘录制逻辑 (模拟)
    function recordShortcut(key) {
        alert(`正在监听 ${key} 的新按键... (逻辑需对接 gohook)`);
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
                                    <label>开机自启</label>
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
                                    <label>唤醒快捷键</label>
                                </div>
                                <button class="shortcut-btn" on:click={() => recordShortcut('wakeUp')}>
                                    {config.shortcuts.wakeUp}
                                </button>
                            </div>
                        </div>
                    {/if}

                    <!-- Tab 3: 外观 -->
                    {#if activeTab === 'appearance'}
                        <div class="setting-group" in:fade={{duration:150}}>
                            <div class="setting-row">
                                <div class="setting-info">
                                    <label>窗口透明度</label>
                                </div>
                                <div class="range-wrapper">
                                    <input type="range" min="0.5" max="1" step="0.05" bind:value={config.appearance.opacity}>
                                    <span>{Math.round(config.appearance.opacity * 100)}%</span>
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
                    <button class="btn-cancel" on:click={close}>取消</button>
                    <button class="btn-save" on:click={save}>保存修改</button>
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
        background: rgba(0, 0, 0, 0.25); /* 轻微遮罩 */
        backdrop-filter: blur(2px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
    }

    .settings-window {
        width: 500px;
        height: 350px; /* 固定高度，保持紧凑 */
        background: #fff;
        border-radius: 8px;
        box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        display: flex;
        overflow: hidden;
        border: 1px solid rgba(0,0,0,0.1);
        font-size: 13px;
    }

    /* --- 侧边栏 --- */
    .sidebar {
        width: 140px;
        background: #f5f5f7;
        border-right: 1px solid #e0e0e0;
        display: flex;
        flex-direction: column;
        padding: 10px 0;
    }

    .sidebar-title {
        padding: 0 16px 10px;
        font-weight: 600;
        color: #888;
        font-size: 12px;
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
        background: #e4e4e7; /* 选中态，不要太刺眼 */
        color: #000;
        font-weight: 500;
        border-left: 3px solid #3b82f6; /* 蓝色指示条 */
    }

    /* --- 内容区 --- */
    .content {
        flex: 1;
        display: flex;
        flex-direction: column;
        background: #fff;
    }

    .content-header {
        padding: 12px 20px;
        border-bottom: 1px solid #f0f0f0;
    }
    .content-header h2 { margin: 0; font-size: 16px; font-weight: 600; color: #333; }

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

    .setting-info label { font-weight: 500; color: #333; margin-bottom: 2px; }
    .setting-info .desc { color: #999; font-size: 12px; }

    /* 快捷键按钮模拟 */
    .shortcut-btn {
        background: #f9f9f9;
        border: 1px solid #ccc;
        border-radius: 4px;
        padding: 4px 10px;
        font-family: monospace;
        font-size: 12px;
        color: #555;
        cursor: pointer;
        min-width: 80px;
    }
    .shortcut-btn:hover { border-color: #999; background: #fff; }

    /* 底部按钮 */
    .content-footer {
        padding: 10px 20px;
        border-top: 1px solid #f0f0f0;
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        background: #fafafa;
    }

    button { font-size: 13px; padding: 6px 14px; border-radius: 4px; cursor: pointer; border: 1px solid transparent; }
    .btn-cancel { background: transparent; color: #666; }
    .btn-cancel:hover { color: #333; background: #e0e0e0; }
    .btn-save { background: #3b82f6; color: white; }
    .btn-save:hover { background: #2563eb; }

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
</style>