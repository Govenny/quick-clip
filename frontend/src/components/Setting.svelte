<script>
    import { createEventDispatcher, onMount } from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { GetConfig, UpdateConfig, RegisterGlobalHotkey, SetOpacity, SetFontSizeLevel } from "../../wailsjs/go/main/App";
    import { ToggleAutoStart, IsAutoStartCheck } from "../../wailsjs/go/internal/AppService";
    import { LogInfo } from '../../wailsjs/runtime/runtime';
    import { internal } from "../../wailsjs/go/models";
    import { applyFontSizeLevel, FONT_SIZE_LEVELS } from "../fontSize";

    const dispatch = createEventDispatcher();

    let config = null;
    let activeTab = 'appearance'; // 'appearance' | 'shortcuts' | 'general'
    const modifiers = ["Alt", "Ctrl", "Shift", "Win"];
    const capsuleModifiers = ["None", "Alt", "Ctrl", "Shift", "Win"];
    const keys = ["Space", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "Return", "Escape", "Delete", "Tab", "Left", "Right", "Up", "Down", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"];
    let selectedMod = "";
    let selectedKey = "";
    let cap1Mod = "Alt";
    let cap1Key = "1";
    let cap2Mod = "Alt";
    let cap2Key = "2";
    let cap3Mod = "Alt";
    let cap3Key = "3";
    let currentFontLevel = 2;

    $: if (config && config.shortcuts) {
        if (!selectedMod && config.shortcuts.wakeUp) {
            selectedMod = config.shortcuts.wakeUp[0] || "Alt";
            selectedKey = config.shortcuts.wakeUp[1] || "Space";
        }
        if (config.shortcuts.capsule1) {
            cap1Mod = config.shortcuts.capsule1[0] || "Alt";
            cap1Key = config.shortcuts.capsule1[1] || "1";
        }
        if (config.shortcuts.capsule2) {
            cap2Mod = config.shortcuts.capsule2[0] || "Alt";
            cap2Key = config.shortcuts.capsule2[1] || "2";
        }
        if (config.shortcuts.capsule3) {
            cap3Mod = config.shortcuts.capsule3[0] || "Alt";
            cap3Key = config.shortcuts.capsule3[1] || "3";
        }
    }

    $: if (config && config.appearance && config.appearance.fontSizeLevel) {
        currentFontLevel = Number(config.appearance.fontSizeLevel);
    }

    $: currentLevelData = FONT_SIZE_LEVELS[Number(currentFontLevel)] || FONT_SIZE_LEVELS[2];

    function selectFontLevel(level) {
        const lvl = Math.max(1, Math.min(5, Number(level) || 2));
        currentFontLevel = lvl;
        if (config && config.appearance) {
            config.appearance.fontSizeLevel = lvl;
        }
        applyFontSizeLevel(lvl);
        SetFontSizeLevel(lvl);
    }

    function updateHotkey() {
        config.shortcuts.wakeUp = [selectedMod, selectedKey];
        LogInfo("新快捷键:" + config.shortcuts.wakeUp);
        RegisterGlobalHotkey(config.shortcuts.wakeUp[0], config.shortcuts.wakeUp[1]);
        UpdateConfig(config);
    }

    function updateCapsuleHotkeys() {
        if (!config || !config.shortcuts) return;
        config.shortcuts.capsule1 = [cap1Mod, cap1Key];
        config.shortcuts.capsule2 = [cap2Mod, cap2Key];
        config.shortcuts.capsule3 = [cap3Mod, cap3Key];
        LogInfo(`胶囊快捷键更新: ${cap1Mod}+${cap1Key}, ${cap2Mod}+${cap2Key}, ${cap3Mod}+${cap3Key}`);
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

    async function syncAutoStart(enabled) {
        try {
            await ToggleAutoStart(enabled);
            LogInfo("开机自启状态已更新:" + enabled);
        } catch (err) {
            LogInfo("设置自启失败:" + err);
        }
    }

    function close() {
        dispatch('close');
    }

    function handleKeydown(e) {
        if (e.key === 'Escape') {
            e.preventDefault();
            close();
        }
    }

    onMount(async () => {
        try {
            const rawConfig = await GetConfig();
            config = internal.Config.createFrom(rawConfig);
            if (config && config.appearance && config.appearance.fontSizeLevel) {
                currentFontLevel = Number(config.appearance.fontSizeLevel);
            }
        } catch (error) {
            console.error('Failed to load config:', error);
        }

        try {
            const isAutoStart = await IsAutoStartCheck();
            if (config && isAutoStart !== config.general.launchAtLogin) {
                config.general.launchAtLogin = isAutoStart;
                UpdateConfig(config);
            }
        } catch (err) {
            console.error("读取自启状态失败:", err);
        }
    });
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="settings-page" in:fly={{ x: 30, duration: 180 }} out:fade={{ duration: 100 }}>
    <!-- 顶栏：克制统一，无多余装饰与强反光 -->
    <header class="settings-header">
        <button class="back-btn" on:click={close} title="返回 (Escape)">
            ← 返回
        </button>
        
        <div class="header-title">偏好设置</div>

        <div class="header-placeholder"></div>
    </header>

    <!-- 顶部轻量分段切换栏：告别全部堆在一坨 -->
    <div class="tabs-nav-wrap">
        <div class="segmented-tabs">
            <button 
                type="button" 
                class="tab-btn" 
                class:active={activeTab === 'appearance'} 
                on:click={() => activeTab = 'appearance'}
            >
                外观
            </button>
            <button 
                type="button" 
                class="tab-btn" 
                class:active={activeTab === 'shortcuts'} 
                on:click={() => activeTab = 'shortcuts'}
            >
                快捷键
            </button>
            <button 
                type="button" 
                class="tab-btn" 
                class:active={activeTab === 'general'} 
                on:click={() => activeTab = 'general'}
            >
                通用
            </button>
        </div>
    </div>

    <!-- 选项卡对应内容区：呼吸感充足，不扎眼，无多余嵌套 -->
    <div class="tab-body">
        {#if config}
            {#if activeTab === 'appearance'}
                <div class="setting-card" in:fade={{ duration: 140 }}>
                    <!-- 字体大小与缩放 -->
                    <div class="setting-col">
                        <div class="setting-header-row">
                            <span class="setting-label">字体与缩放</span>
                            <span class="setting-tag">{currentLevelData.name} · {currentLevelData.fontSize}</span>
                        </div>
                        
                        <!-- 微信同款 5 档滑块 -->
                        <div class="wechat-slider-box">
                            <div class="slider-track-wrap">
                                <div class="track-line"></div>
                                {#each [1, 2, 3, 4, 5] as lvl}
                                    <button 
                                        type="button"
                                        class="step-mark" 
                                        class:active={Number(currentFontLevel) === lvl}
                                        style="left: {(lvl - 1) * 25}%"
                                        on:click={() => selectFontLevel(lvl)}
                                        aria-label={`选择 ${FONT_SIZE_LEVELS[lvl].name} 档位`}
                                    >
                                        <span class="mark-dot"></span>
                                    </button>
                                {/each}
                                <div 
                                    class="slider-thumb" 
                                    style="left: {(Number(currentFontLevel) - 1) * 25}%"
                                >
                                    <div class="thumb-core"></div>
                                </div>
                            </div>
                            <div class="step-labels">
                                {#each [1, 2, 3, 4, 5] as lvl}
                                    <button 
                                        type="button" 
                                        class="step-label-btn" 
                                        class:active={Number(currentFontLevel) === lvl}
                                        on:click={() => selectFontLevel(lvl)}
                                    >
                                        {FONT_SIZE_LEVELS[lvl].name}
                                    </button>
                                {/each}
                            </div>
                        </div>

                        <!-- 极简行内预览：无冗余边框与嵌套 -->
                        <div class="sample-preview" style="font-size: {currentLevelData.fontSize};">
                            <span class="sample-badge">常用</span>
                            <span class="sample-text">示例条目五字排版效果</span>
                        </div>
                    </div>

                    <div class="setting-divider"></div>

                    <!-- 窗口透明度 -->
                    <div class="setting-row">
                        <div>
                            <div class="setting-label">窗口透明度</div>
                            <div class="setting-desc">调节毛玻璃界面的整体透度</div>
                        </div>
                        <div class="range-field">
                            <input 
                                type="range" 
                                class="styled-range"
                                min="25" 
                                max="255" 
                                step="1" 
                                bind:value={config.appearance.opacity}
                                on:change={updateOpacity}
                            >
                            <span class="range-val">{Math.round((config.appearance.opacity / 255) * 100)}%</span>
                        </div>
                    </div>
                </div>

            {:else if activeTab === 'shortcuts'}
                <div class="setting-card" in:fade={{ duration: 140 }}>
                    <!-- 唤醒快捷键 -->
                    <div class="setting-row">
                        <div>
                            <div class="setting-label">全局呼出按键</div>
                            <div class="setting-desc">快速呼出主界面快捷键</div>
                        </div>
                        <div class="hotkey-wrapper">
                            <select class="styled-select" bind:value={selectedMod} on:change={updateHotkey}>
                                {#each modifiers as mod}
                                    <option value={mod}>{mod}</option>
                                {/each}
                            </select>
                            <span class="hotkey-plus">+</span>
                            <select class="styled-select" bind:value={selectedKey} on:change={updateHotkey}>
                                {#each keys as key}
                                    <option value={key}>{key}</option>
                                {/each}
                            </select>
                        </div>
                    </div>
                    <div class="setting-divider"></div>

                    <!-- 胶囊 1 快捷键 -->
                    <div class="setting-row">
                        <div>
                            <div class="setting-label">胶囊 1 快捷键</div>
                            <div class="setting-desc">多胶囊时首个常用胶囊快捷键</div>
                        </div>
                        <div class="hotkey-wrapper">
                            <select class="styled-select" bind:value={cap1Mod} on:change={updateCapsuleHotkeys}>
                                {#each capsuleModifiers as mod}
                                    <option value={mod}>{mod === 'None' ? '无' : mod}</option>
                                {/each}
                            </select>
                            <span class="hotkey-plus">+</span>
                            <select class="styled-select" bind:value={cap1Key} on:change={updateCapsuleHotkeys}>
                                {#each keys as key}
                                    <option value={key}>{key}</option>
                                {/each}
                            </select>
                        </div>
                    </div>

                    <div class="setting-divider"></div>

                    <!-- 胶囊 2 快捷键 -->
                    <div class="setting-row">
                        <div>
                            <div class="setting-label">胶囊 2 快捷键</div>
                            <div class="setting-desc">多胶囊时第二常用胶囊快捷键</div>
                        </div>
                        <div class="hotkey-wrapper">
                            <select class="styled-select" bind:value={cap2Mod} on:change={updateCapsuleHotkeys}>
                                {#each capsuleModifiers as mod}
                                    <option value={mod}>{mod === 'None' ? '无' : mod}</option>
                                {/each}
                            </select>
                            <span class="hotkey-plus">+</span>
                            <select class="styled-select" bind:value={cap2Key} on:change={updateCapsuleHotkeys}>
                                {#each keys as key}
                                    <option value={key}>{key}</option>
                                {/each}
                            </select>
                        </div>
                    </div>

                    <div class="setting-divider"></div>

                    <!-- 胶囊 3 快捷键 -->
                    <div class="setting-row">
                        <div>
                            <div class="setting-label">胶囊 3 快捷键</div>
                            <div class="setting-desc">多胶囊时第三常用胶囊快捷键</div>
                        </div>
                        <div class="hotkey-wrapper">
                            <select class="styled-select" bind:value={cap3Mod} on:change={updateCapsuleHotkeys}>
                                {#each capsuleModifiers as mod}
                                    <option value={mod}>{mod === 'None' ? '无' : mod}</option>
                                {/each}
                            </select>
                            <span class="hotkey-plus">+</span>
                            <select class="styled-select" bind:value={cap3Key} on:change={updateCapsuleHotkeys}>
                                {#each keys as key}
                                    <option value={key}>{key}</option>
                                {/each}
                            </select>
                        </div>
                    </div>

                    <div class="setting-divider"></div>

                    <!-- 粘贴缓冲时间 -->
                    <div class="setting-row">
                        <div>
                            <div class="setting-label">粘贴缓冲延时</div>
                            <div class="setting-desc">模拟发送粘贴的安全缓冲时间</div>
                        </div>
                        <div class="range-field">
                            <input 
                                type="range" 
                                class="styled-range"
                                min="25" 
                                max="800" 
                                step="25" 
                                bind:value={config.shortcuts.pasteWaitTime}
                                on:change={updatePasteWaitTime}
                            >
                            <span class="range-val">{config.shortcuts.pasteWaitTime}ms</span>
                        </div>
                    </div>
                </div>

            {:else if activeTab === 'general'}
                <div class="setting-card" in:fade={{ duration: 140 }}>
                    <div class="setting-row">
                        <div>
                            <div class="setting-label">开机自启</div>
                            <div class="setting-desc">登录 Windows 时自动在后台运行</div>
                        </div>
                        <label class="toggle-switch">
                            <input 
                                type="checkbox" 
                                bind:checked={config.general.launchAtLogin} 
                                on:change={() => syncAutoStart(config.general.launchAtLogin)}
                            >
                            <span class="toggle-track">
                                <span class="toggle-thumb"></span>
                            </span>
                        </label>
                    </div>
                </div>

                <div class="about-card" in:fade={{ duration: 140 }}>
                    <div class="about-name">Quick-Clip</div>
                    <div class="about-meta">版本 1.2.0 · 本地安全</div>
                    <div class="about-desc">轻量、极速、场景自学习的桌面剪贴板工具</div>
                </div>
            {/if}
        {/if}
    </div>
</div>

<style>
    * { box-sizing: border-box; }

    .settings-page {
        width: 100%;
        height: 100%;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        background: transparent;
        user-select: none;
    }

    /* --- 顶栏：克制无反光 --- */
    .settings-header {
        flex-shrink: 0;
        background: rgba(244, 248, 251, 0.45);
        border-bottom: 1px solid rgba(74, 96, 116, 0.08);
        padding: 0 12px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        height: 40px;
    }

    .back-btn {
        display: inline-flex;
        align-items: center;
        background: rgba(255, 255, 255, 0.45);
        border: 1px solid rgba(148, 163, 184, 0.26);
        border-radius: 5px;
        padding: 3px 9px;
        color: #475569;
        font-size: 12px;
        font-weight: 500;
        cursor: pointer;
        outline: none;
        transition: all 0.15s ease;
    }

    .back-btn:hover {
        background: rgba(255, 255, 255, 0.8);
        color: #1e293b;
        border-color: rgba(100, 116, 139, 0.38);
    }

    .header-title {
        font-size: 13px;
        font-weight: 600;
        color: #334155;
        letter-spacing: 0.2px;
    }

    .header-placeholder {
        width: 50px;
    }

    /* --- 分段导航栏：分流布局，告别一坨乱 --- */
    .tabs-nav-wrap {
        padding: 10px 14px 4px;
        flex-shrink: 0;
    }

    .segmented-tabs {
        display: flex;
        background: rgba(148, 163, 184, 0.15);
        border-radius: 7px;
        padding: 2px;
        gap: 2px;
    }

    .tab-btn {
        flex: 1;
        text-align: center;
        padding: 5px 0;
        font-size: 12px;
        font-weight: 500;
        color: #64748b;
        border: none;
        background: transparent;
        border-radius: 5px;
        cursor: pointer;
        transition: all 0.15s ease;
        outline: none;
    }

    .tab-btn:hover {
        color: #334155;
    }

    .tab-btn.active {
        background: rgba(255, 255, 255, 0.82);
        color: #1e293b;
        font-weight: 600;
        box-shadow: 0 1px 3px rgba(15, 23, 42, 0.07);
    }

    /* --- 内容区 --- */
    .tab-body {
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        padding: 10px 14px 20px;
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .tab-body::-webkit-scrollbar {
        width: 4px;
    }
    .tab-body::-webkit-scrollbar-thumb {
        background: rgba(100, 116, 139, 0.15);
        border-radius: 4px;
    }

    /* --- 柔和卡片：无刺眼强白反光 --- */
    .setting-card {
        background: rgba(255, 255, 255, 0.46);
        border: 1px solid rgba(255, 255, 255, 0.65);
        border-radius: 8px;
        padding: 14px;
        box-shadow: 0 1px 2px rgba(15, 23, 42, 0.03);
    }

    .setting-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
    }

    .setting-col {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .setting-header-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
    }

    .setting-label {
        font-size: 13px;
        font-weight: 550;
        color: #1e293b;
    }

    .setting-desc {
        font-size: 11px;
        color: #64748b;
        margin-top: 2px;
    }

    .setting-tag {
        font-size: 11px;
        font-weight: 550;
        padding: 1px 6px;
        background: rgba(71, 85, 105, 0.08);
        color: #475569;
        border-radius: 4px;
    }

    .setting-divider {
        height: 1px;
        background: rgba(71, 85, 105, 0.07);
        margin: 14px 0;
    }

    /* --- 微信滑块：低饱和、柔和雅致 --- */
    .wechat-slider-box {
        display: flex;
        flex-direction: column;
        width: 100%;
        margin-top: 4px;
    }

    .slider-track-wrap {
        position: relative;
        width: 100%;
        height: 28px;
        display: flex;
        align-items: center;
    }

    .track-line {
        position: absolute;
        left: 0;
        right: 0;
        height: 3px;
        background: rgba(148, 163, 184, 0.3);
        border-radius: 2px;
        z-index: 1;
    }

    .step-mark {
        position: absolute;
        top: 50%;
        transform: translate(-50%, -50%);
        width: 28px;
        height: 28px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        background: transparent !important;
        border: none !important;
        padding: 0 !important;
        z-index: 2;
    }

    .mark-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: #94a3b8;
        transition: transform 0.18s ease, background-color 0.18s ease;
    }

    .step-mark:hover .mark-dot {
        background: #475569;
        transform: scale(1.3);
    }

    .step-mark.active .mark-dot {
        background: #334155;
        transform: scale(1.3);
    }

    .slider-thumb {
        position: absolute;
        top: 50%;
        transform: translate(-50%, -50%);
        width: 18px;
        height: 18px;
        border-radius: 50%;
        background: #ffffff;
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.18), 0 0 0 1px rgba(0, 0, 0, 0.06);
        display: flex;
        align-items: center;
        justify-content: center;
        pointer-events: none;
        z-index: 3;
        transition: left 0.22s ease;
    }

    .thumb-core {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: #475569;
    }

    .step-labels {
        display: flex;
        justify-content: space-between;
        width: 100%;
        margin-top: 1px;
    }

    .step-label-btn {
        border: none !important;
        background: transparent !important;
        font-size: 11px;
        color: #64748b;
        cursor: pointer;
        padding: 2px 4px !important;
        border-radius: 4px;
        transition: color 0.15s ease;
    }

    .step-label-btn:hover {
        color: #1e293b;
    }

    .step-label-btn.active {
        color: #1e293b;
        font-weight: 600;
    }

    /* --- 行内轻量预览 --- */
    .sample-preview {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 7px 10px;
        margin-top: 4px;
        background: rgba(241, 245, 249, 0.45);
        border: 1px solid rgba(148, 163, 184, 0.18);
        border-radius: 6px;
        color: #334155;
    }

    .sample-badge {
        font-size: 10px;
        font-weight: 550;
        padding: 1px 5px;
        background: rgba(71, 85, 105, 0.1);
        color: #475569;
        border-radius: 3px;
        flex-shrink: 0;
    }

    .sample-text {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-weight: 500;
    }

    /* --- 控件：范围滑块、下拉选择、Toggle 开关 --- */
    .range-field {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-shrink: 0;
    }

    .styled-range {
        width: 90px;
        accent-color: #475569;
        cursor: pointer;
    }

    .range-val {
        font-size: 12px;
        font-weight: 500;
        color: #475569;
        min-width: 42px;
        text-align: right;
    }

    .hotkey-wrapper {
        display: flex;
        align-items: center;
        gap: 4px;
        flex-shrink: 0;
    }

    .styled-select {
        appearance: none;
        background: rgba(255, 255, 255, 0.6);
        border: 1px solid rgba(148, 163, 184, 0.3);
        border-radius: 5px;
        padding: 4px 8px;
        font-size: 12px;
        font-weight: 500;
        color: #1e293b;
        outline: none;
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .styled-select:hover {
        background: rgba(255, 255, 255, 0.85);
        border-color: rgba(100, 116, 139, 0.45);
    }

    .styled-select:focus {
        border-color: #475569;
    }

    .hotkey-plus {
        font-size: 12px;
        font-weight: 500;
        color: #94a3b8;
    }

    .toggle-switch {
        position: relative;
        width: 36px;
        height: 20px;
        display: inline-block;
        flex-shrink: 0;
        cursor: pointer;
    }

    .toggle-switch input {
        opacity: 0;
        width: 0;
        height: 0;
        margin: 0;
    }

    .toggle-track {
        position: absolute;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(148, 163, 184, 0.35);
        border-radius: 20px;
        transition: background-color 0.2s ease;
    }

    .toggle-thumb {
        position: absolute;
        top: 2px;
        left: 2px;
        width: 16px;
        height: 16px;
        background: #ffffff;
        border-radius: 50%;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
        transition: transform 0.2s ease;
    }

    .toggle-switch input:checked + .toggle-track {
        background: #475569;
    }

    .toggle-switch input:checked + .toggle-track .toggle-thumb {
        transform: translateX(16px);
    }

    /* 关于卡片 */
    .about-card {
        background: rgba(255, 255, 255, 0.35);
        border: 1px solid rgba(255, 255, 255, 0.5);
        border-radius: 8px;
        padding: 14px;
        text-align: center;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 3px;
    }

    .about-name {
        font-size: 14px;
        font-weight: 650;
        color: #1e293b;
    }

    .about-meta {
        font-size: 11px;
        color: #64748b;
    }

    .about-desc {
        font-size: 11px;
        color: #94a3b8;
        margin-top: 4px;
    }
</style>