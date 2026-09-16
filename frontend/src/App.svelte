<script>
    import { onMount, tick, onDestroy } from 'svelte';
    import { fade, fly, slide } from 'svelte/transition';
    import { cubicOut } from 'svelte/easing';

    // iOS 弹性缓动: cubic-bezier(0.34, 1.3, 0.64, 1)
    function iosElastic(t) {
        const p1x = 0.34, p1y = 1.3, p2x = 0.64, p2y = 1;
        let u = t;
        for (let i = 0; i < 8; i++) {
            const b = 3*p1x*u*(1-u)*(1-u) + 3*p2x*u*u*(1-u) + u*u*u;
            const db = 3*p1x*(1-u)*(1-u) - 6*p1x*u*(1-u) + 3*p2x*u*(2-3*u) + 3*u*u;
            if (Math.abs(b - t) < 1e-5) break;
            u -= (b - t) / db;
        }
        return 3*p1y*u*(1-u)*(1-u) + 3*p2y*u*u*(1-u) + u*u*u;
    }
    import { EnterSettingsMode, GetConfig, GetContent, SaveContent, ExitSettingsMode, ToggleWindow, HideWindow, GetContextSuggestions, GetContextSlots, PinSlot, UnpinSlot, RecordItemUsage, RemoveItemUsage } from '../wailsjs/go/main/App'; 
    import { applyFontSizeLevel } from './fontSize';
    import { LogInfo, EventsOn } from '../wailsjs/runtime';
    import TreeItem from './components/TreeItem.svelte';
    import Setting from './components/Setting.svelte';
    import ContextMenu from './components/ContextMenu.svelte';
    import TextModal from './components/TextModal.svelte';
    import DirModal from './components/DirModal.svelte';
    import ConfirmModal from './components/ConfirmModal.svelte';
    import { normalizeTree, deleteNodeById, moveNode, searchTree, generateId, findNodeById } from './utils/treeAdapter';

    let data = [];
    let expanded = {};
    let showMenu = false;
    let showSettings = false;

    // 3 个常用槽位状态 (Slot 1, 2, 3)
    let capsuleSlots = [
        { slot: 1, type: 'empty', item: null },
        { slot: 2, type: 'empty', item: null },
        { slot: 3, type: 'empty', item: null }
    ];

    $: activeSlots = capsuleSlots.filter(s => s && s.type !== 'empty' && s.item);
    $: activeSlotsCount = activeSlots.length;
    $: hasActiveSlots = activeSlotsCount > 0;

    // 槽位替换确认弹窗状态
    let showSlotConfirm = false;
    let pendingSlotTarget = { slot: 1, newNode: null, currentItem: null };

    // 粘贴模式开关: true=Auto Paste, false=Not Paste
    let autoPaste = true;
    let isProcessingPaste = false;

    // 当前编辑/父级节点
    let editingNode = null;
    let targetFolderNode = null;

    // 目录弹窗状态
    let showDirInput = false;
    let isEditDirMode = false;
    let dirModalName = "";

    // 文本弹窗状态
    let showTextInput = false;
    let isEditTextMode = false;
    let textModalTitle = "";
    let textModalValue = "";

    // 胶囊快捷键配置 (胶囊 1, 2, 3)
    let capsuleShortcuts = [
        ["Alt", "1"],
        ["Alt", "2"],
        ["Alt", "3"]
    ];

    function syncCapsuleShortcutsFromConfig(cfg) {
        if (cfg && cfg.shortcuts) {
            if (cfg.shortcuts.capsule1 && cfg.shortcuts.capsule1[1]) capsuleShortcuts[0] = cfg.shortcuts.capsule1;
            if (cfg.shortcuts.capsule2 && cfg.shortcuts.capsule2[1]) capsuleShortcuts[1] = cfg.shortcuts.capsule2;
            if (cfg.shortcuts.capsule3 && cfg.shortcuts.capsule3[1]) capsuleShortcuts[2] = cfg.shortcuts.capsule3;
            capsuleShortcuts = [...capsuleShortcuts];
        }
    }

    // 删除确认弹窗
    let showDeleteConfirm = false;
    let itemToDelete = null;
    let deleteType = 'item'; // 'item' | 'capsule'

    // 全局上下文菜单
    let globalContextMenu = {
        visible: false,
        x: 0,
        y: 0,
        targetNode: null,
        isCapsule: false,
        capsuleSlot: null,
        flipX: false,
        flipY: false
    };

    function cleanGlobalContextMenu() {
        globalContextMenu = {
            visible: false,
            x: 0,
            y: 0,
            targetNode: null,
            isCapsule: false,
            capsuleSlot: null,
            flipX: false,
            flipY: false
        };
    }

    function showContextMenu(e, node) {
        e.preventDefault();
        e.stopPropagation();
        
        const isFolder = node && node.type === 'folder';
        const menuWidth = 148;
        const itemHeight = 28;
        const dividerHeight = 9;
        const padding = 8;
        
        let itemCount, dividerCount;
        if (isFolder) {
            itemCount = 4; // New Text + New Folder + Edit + Delete
            dividerCount = 2;
        } else {
            itemCount = 3; // 固定至常用槽位 + Edit + Delete
            dividerCount = 2;
        }
        const menuHeight = itemCount * itemHeight + dividerCount * dividerHeight + padding;
        
        const winW = window.innerWidth;
        const winH = window.innerHeight;
        
        const flipX = e.pageX + menuWidth > winW;
        const flipY = e.pageY + menuHeight > winH;
        
        globalContextMenu = {
            visible: true,
            x: flipX ? e.pageX - menuWidth : e.pageX,
            y: flipY ? e.pageY - menuHeight : e.pageY,
            targetNode: node,
            isCapsule: false,
            capsuleSlot: null,
            flipX,
            flipY
        };
    }

    function handleCapsuleContextMenu(e, slot) {
        e.preventDefault();
        e.stopPropagation();
        if (!slot || slot.type === 'empty' || !slot.item) return;

        const menuWidth = 148;
        const itemHeight = 28;
        const dividerHeight = 9;
        const padding = 8;
        const itemCount = slot.type === 'pinned' ? 3 : 4;
        const dividerCount = 2;
        const menuHeight = itemCount * itemHeight + dividerCount * dividerHeight + padding;

        const winW = window.innerWidth;
        const winH = window.innerHeight;

        const flipX = e.pageX + menuWidth > winW;
        const flipY = e.pageY + menuHeight > winH;

        globalContextMenu = {
            visible: true,
            x: flipX ? e.pageX - menuWidth : e.pageX,
            y: flipY ? e.pageY - menuHeight : e.pageY,
            targetNode: slot.item,
            isCapsule: true,
            capsuleSlot: slot,
            flipX,
            flipY
        };
    }

    function hideContextMenu() {
        globalContextMenu.visible = false;
    }

    function handleGlobalClick(e) {
        if (!e.target.closest('.context-menu')) {
            hideContextMenu();
        }

        if (!e.target.closest('.dropdown-menu') && !e.target.closest('.add-btn') && !e.target.closest('.icon-btn')) {
            cancelMenu();
        }
    }

    function removeCapsule() {
        if (!globalContextMenu.targetNode) return;
        itemToDelete = globalContextMenu.targetNode;
        deleteType = 'capsule';
        showDeleteConfirm = true;
        hideContextMenu();
    }

    function deleteItem() {
        if (!globalContextMenu.targetNode) return;
        itemToDelete = globalContextMenu.targetNode;
        deleteType = 'item';
        showDeleteConfirm = true;
        hideContextMenu();
    }

    async function confirmDeleteItem() {
        if (!itemToDelete) return;

        if (deleteType === 'capsule') {
            try {
                await RemoveItemUsage(itemToDelete.id);
                await loadSuggestions();
                cancelDelete();
            } catch (err) {
                console.error("移除胶囊失败", err);
            }
        } else {
            try {
                deleteNodeById(data, itemToDelete.id);
                updateData([...data]);
                await RemoveItemUsage(itemToDelete.id);
                await loadSuggestions();
                cancelDelete();
            } catch (err) {
                console.error("删除失败", err);
            }
        }
    }

    function cancelDelete() {
        showDeleteConfirm = false;
        itemToDelete = null;
        cleanGlobalContextMenu();
    }

    async function handleAssignSlot(event) {
        const { slot, targetNode, currentSlot } = event.detail;
        if (!targetNode) return;

        if (!currentSlot || currentSlot.type === 'empty' || !currentSlot.item) {
            // 空槽直接绑定，无需二次确认
            try {
                await PinSlot(slot, targetNode.id);
                await loadSuggestions();
            } catch (err) {
                console.error("固定槽位失败", err);
            }
            hideContextMenu();
        } else {
            // 目标槽位已被占用：弹出二次确认
            pendingSlotTarget = {
                slot,
                newNode: targetNode,
                currentItem: currentSlot.item
            };
            showSlotConfirm = true;
            hideContextMenu();
        }
    }

    async function confirmSlotReplace() {
        if (!pendingSlotTarget.newNode) return;
        try {
            await PinSlot(pendingSlotTarget.slot, pendingSlotTarget.newNode.id);
            await loadSuggestions();
        } catch (err) {
            console.error("替换槽位失败", err);
        } finally {
            showSlotConfirm = false;
            pendingSlotTarget = { slot: 1, newNode: null, currentItem: null };
            cleanGlobalContextMenu();
        }
    }

    function cancelSlotReplace() {
        showSlotConfirm = false;
        pendingSlotTarget = { slot: 1, newNode: null, currentItem: null };
        cleanGlobalContextMenu();
    }

    async function handleUnpinSlot(event) {
        const slotNum = event.detail;
        try {
            await UnpinSlot(slotNum);
            await loadSuggestions();
        } catch (err) {
            console.error("取消固定槽位失败", err);
        }
        hideContextMenu();
    }

    async function handlePinHere(event) {
        const slot = event.detail;
        if (!slot || !slot.item) return;
        try {
            await PinSlot(slot.slot, slot.item.id);
            await loadSuggestions();
        } catch (err) {
            console.error("锁定槽位失败", err);
        }
        hideContextMenu();
    }

    // 监听来自后端的 show-settings 事件
    const settingsEventListener = async () => {
        showSettings = true;
        EnterSettingsMode();
        ToggleWindow();
    };

    function openSettings() {
        showMenu = false;
        showSettings = true;
        EnterSettingsMode();
    }

    async function closeSettings() {
        showSettings = false;
        ExitSettingsMode();
        try {
            const cfg = await GetConfig();
            syncCapsuleShortcutsFromConfig(cfg);
        } catch (e) {}
    }

    // 监听来自后端的 update-content 事件
    const contentEventListener = async () => {
        try {
            LogInfo("update-content发送成功");
            const newData = await GetContent();
            data = normalizeTree(newData);
            await loadSuggestions();
            await tick();
        } catch (error) {
            console.error('Failed to load content:', error);
        }
    };

    async function loadSuggestions() {
        try {
            const rawSlots = await GetContextSlots();
            if (rawSlots && rawSlots.length > 0) {
                capsuleSlots = rawSlots.map(s => {
                    if (s.itemId) {
                        const node = findNodeById(data, s.itemId);
                        if (node && node.type === 'text') {
                            return { slot: s.slot, type: s.type, item: node };
                        }
                    }
                    return { slot: s.slot, type: 'empty', item: null };
                });
            } else {
                capsuleSlots = [
                    { slot: 1, type: 'empty', item: null },
                    { slot: 2, type: 'empty', item: null },
                    { slot: 3, type: 'empty', item: null }
                ];
            }
        } catch (err) {
            console.error("加载槽位推荐失败", err);
            capsuleSlots = [
                { slot: 1, type: 'empty', item: null },
                { slot: 2, type: 'empty', item: null },
                { slot: 3, type: 'empty', item: null }
            ];
        }
    }

    onMount(() => {
        document.addEventListener('click', handleGlobalClick);
        document.addEventListener('contextmenu', hideContextMenu);

        EventsOn("show-settings", settingsEventListener);
        EventsOn("update-content", contentEventListener);
        EventsOn("window-shown", async () => {
            isProcessingPaste = false;
            await loadSuggestions();
        });
    });

    onMount(async () => {
        try {
            const cfg = await GetConfig();
            syncCapsuleShortcutsFromConfig(cfg);
            if (cfg && cfg.appearance && cfg.appearance.fontSizeLevel) {
                applyFontSizeLevel(cfg.appearance.fontSizeLevel);
            }
            const rawData = await GetContent();
            data = normalizeTree(rawData);
            await loadSuggestions();
        } catch (error) {
            console.error('Failed to load content/config:', error);
        }
    });

    onDestroy(() => {
        document.removeEventListener('click', handleGlobalClick);
        document.removeEventListener('contextmenu', hideContextMenu);
    });

    function toggleExpand(id) {
        expanded[id] = !expanded[id];
        expanded = expanded;
    }

    function toggleMenu() {
        showMenu = !showMenu;
        if (showMenu) {
            targetFolderNode = null;
        }
    }

    function addText() {
        isEditTextMode = false;
        editingNode = null;
        targetFolderNode = (globalContextMenu.targetNode && globalContextMenu.targetNode.type === 'folder')
            ? globalContextMenu.targetNode
            : null;
        textModalTitle = "";
        textModalValue = "";
        showTextInput = true;
        showMenu = false;
        hideContextMenu();
    }

    function editText() {
        if (!globalContextMenu.targetNode) return;
        isEditTextMode = true;
        editingNode = globalContextMenu.targetNode;
        targetFolderNode = null;
        textModalTitle = editingNode.name;
        textModalValue = editingNode.value || "";
        showTextInput = true;
        showMenu = false;
        hideContextMenu();
    }

    function handleTextSubmit(event) {
        const { title, value } = event.detail;
        if (isEditTextMode && editingNode) {
            editingNode.name = title;
            editingNode.value = value;
            updateData([...data]);
        } else {
            const newNode = {
                id: generateId(),
                name: title,
                type: 'text',
                value: value
            };

            if (targetFolderNode && targetFolderNode.type === 'folder') {
                if (!Array.isArray(targetFolderNode.children)) {
                    targetFolderNode.children = [];
                }
                targetFolderNode.children.push(newNode);
                expanded[targetFolderNode.id] = true;
                expanded = expanded;
            } else {
                data = [...data, newNode];
            }
            updateData([...data]);
        }
        closeTextModal();
    }

    function closeTextModal() {
        showTextInput = false;
        isEditTextMode = false;
        editingNode = null;
        targetFolderNode = null;
        cleanGlobalContextMenu();
    }

    function addDir() {
        isEditDirMode = false;
        editingNode = null;
        targetFolderNode = (globalContextMenu.targetNode && globalContextMenu.targetNode.type === 'folder')
            ? globalContextMenu.targetNode
            : null;
        dirModalName = "";
        showDirInput = true;
        showMenu = false;
        hideContextMenu();
    }

    function editDir() {
        if (!globalContextMenu.targetNode) return;
        isEditDirMode = true;
        editingNode = globalContextMenu.targetNode;
        targetFolderNode = null;
        dirModalName = editingNode.name;
        showDirInput = true;
        showMenu = false;
        hideContextMenu();
    }

    function handleDirSubmit(event) {
        const { name } = event.detail;
        if (isEditDirMode && editingNode) {
            editingNode.name = name;
            updateData([...data]);
        } else {
            const newNode = {
                id: generateId(),
                name: name,
                type: 'folder',
                children: []
            };

            if (targetFolderNode && targetFolderNode.type === 'folder') {
                if (!Array.isArray(targetFolderNode.children)) {
                    targetFolderNode.children = [];
                }
                targetFolderNode.children.push(newNode);
                expanded[targetFolderNode.id] = true;
                expanded = expanded;
            } else {
                data = [...data, newNode];
            }
            updateData([...data]);
        }
        closeDirModal();
    }

    function closeDirModal() {
        showDirInput = false;
        isEditDirMode = false;
        editingNode = null;
        targetFolderNode = null;
        cleanGlobalContextMenu();
    }

    function cancelMenu() {
        showMenu = false;
    }

    function handleMoveNode(sourceId, targetId, dropType) {
        const success = moveNode(data, sourceId, targetId, dropType);
        if (success) {
            if (dropType === 'inside') {
                expanded[targetId] = true;
                expanded = expanded;
            }
            updateData([...data]);
        }
    }

    // 焦点--------------------------------------------
    function handleBlur() {
        setTimeout(() => {
            if (isProcessingPaste) {
                return;
            }

            if (document.hasFocus()) {
                return;
            }

            if (showTextInput || showDirInput || showSettings) {
                return; 
            }

            if (showDeleteConfirm) {
                cancelDelete();
            }
            if (showSlotConfirm) {
                cancelSlotReplace();
            }

            HideWindow();
        }, 10);
    }

    function updateData(newData) {
        data = newData;
        SaveContent(data);
    }

    import { PasteAndHide, HideAndRestore } from '../wailsjs/go/main/App';
    let searchQuery = "";
    let searchResults = [];

    $: {
        if (searchQuery.trim()) {
            searchResults = searchTree(data, searchQuery);
        } else {
            searchResults = [];
        }
    }

    function handleSlotClick(slot) {
        if (!slot || slot.type === 'empty' || !slot.item) return;
        const item = slot.item;
        if (item.id) {
            RecordItemUsage(item.id);
        }
        isProcessingPaste = true;
        const content = item.value || '';
        navigator.clipboard.writeText(content).then(() => {
            if (autoPaste) {
                PasteAndHide();
            } else {
                HideAndRestore();
            }
        }).catch(err => {
            console.error("Slot copy failed:", err);
            if (autoPaste) {
                PasteAndHide();
            } else {
                HideAndRestore();
            }
        });
    }

    function handleSearchResultClick(result) {
        if (result && result.id) {
            RecordItemUsage(result.id);
        }
        isProcessingPaste = true;
        const content = typeof result === 'string' ? result : (result.content || '');
        navigator.clipboard.writeText(content).then(() => {
            if (autoPaste) {
                PasteAndHide();
            } else {
                HideAndRestore();
            }
            searchQuery = "";
        }).catch(err => {
            console.error("Search copy failed:", err);
            if (autoPaste) {
                PasteAndHide();
            } else {
                HideAndRestore();
            }
        });
    }

    function getCapsuleBadgeLabel(idx) {
        const shortcut = capsuleShortcuts[idx];
        if (!shortcut || !Array.isArray(shortcut)) return '';
        const [mod, key] = shortcut;
        if (!mod || mod === 'None') {
            return key;
        }
        return `${mod}+${key}`;
    }

    function isEnterKey(shortcut) {
        if (!shortcut || !Array.isArray(shortcut)) return false;
        const key = (shortcut[1] || '').toLowerCase();
        return key === 'enter' || key === 'return';
    }

    function matchesShortcut(event, shortcut) {
        if (!shortcut || !Array.isArray(shortcut) || shortcut.length < 2) return false;
        const [mod, key] = shortcut;
        if (!key) return false;

        const alt = mod === 'Alt';
        const ctrl = mod === 'Ctrl';
        const shift = mod === 'Shift';
        const win = mod === 'Win';
        const noMod = !mod || mod === 'None' || mod === '';

        if (noMod) {
            if (event.altKey || event.ctrlKey || event.shiftKey || event.metaKey) return false;
            const tag = document.activeElement?.tagName;
            if (tag === 'INPUT' || tag === 'TEXTAREA') return false;
        } else {
            if (alt && !event.altKey) return false;
            if (ctrl && !event.ctrlKey) return false;
            if (shift && !event.shiftKey) return false;
            if (win && !event.metaKey) return false;
        }

        const eventKey = (event.key || '').toLowerCase();
        const targetKey = key.toLowerCase();

        if (targetKey === 'space') {
            return eventKey === ' ' || eventKey === 'space' || event.code === 'Space';
        }
        if (targetKey === 'return' || targetKey === 'enter') {
            return eventKey === 'enter';
        }
        if (targetKey === 'escape') {
            return eventKey === 'escape';
        }

        return eventKey === targetKey || event.code === `Key${key.toUpperCase()}` || event.code === `Digit${key}`;
    }

    function handleGlobalKeydown(e) {
        if (showTextInput || showDirInput || showDeleteConfirm || showSlotConfirm || showSettings) {
            return;
        }
        if (document.activeElement && document.activeElement.classList.contains('paste-toggle')) {
            return;
        }

        // 仅当且仅当恰好只有 1 个有效常用胶囊时，敲回车直接触发
        if (activeSlotsCount === 1 && !searchQuery.trim()) {
            if (e.key === 'Enter') {
                e.preventDefault();
                handleSlotClick(activeSlots[0]);
                return;
            }
        }

        // 匹配 3 个槽位的独立快捷键 (如 Alt+1, Alt+2, Alt+3, None+Enter 等)
        for (let i = 0; i < 3; i++) {
            if (matchesShortcut(e, capsuleShortcuts[i])) {
                const slot = capsuleSlots[i];
                if (slot && slot.type !== 'empty' && slot.item) {
                    e.preventDefault();
                    handleSlotClick(slot);
                    return;
                }
            }
        }
    }

    function handleSearchKeydown(e) {
        if (e.key === 'Enter') {
            if (searchQuery.trim()) {
                if (searchResults.length > 0) {
                    e.preventDefault();
                    handleSearchResultClick(searchResults[0]);
                }
            } else {
                // 搜索框为空时按回车：
                // 1. 若恰好只有 1 个有效胶囊，默认直接触发该有效胶囊
                if (activeSlotsCount === 1) {
                    e.preventDefault();
                    handleSlotClick(activeSlots[0]);
                    return;
                }
                // 2. 检查是否有胶囊被配置为无修饰键回车 (None + Enter)
                for (let i = 0; i < 3; i++) {
                    const sc = capsuleShortcuts[i];
                    if (sc && (!sc[0] || sc[0] === 'None') && (sc[1]?.toLowerCase() === 'enter' || sc[1]?.toLowerCase() === 'return')) {
                        const slot = capsuleSlots[i];
                        if (slot && slot.type !== 'empty' && slot.item) {
                            e.preventDefault();
                            handleSlotClick(slot);
                            return;
                        }
                    }
                }
            }
        }
    }
</script>

<svelte:window 
    on:blur={() => handleBlur()} 
    on:keydown={handleGlobalKeydown}
/>

<div class="app-container">
    {#if showSettings}
        <Setting on:close={closeSettings} />
    {:else}
        <div class="sticky-header">
            <div class="header-row">
                <button class="paste-toggle" class:active={autoPaste} on:click={() => { autoPaste = !autoPaste; }} on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); autoPaste = !autoPaste; } }}>
                    <span class="toggle-text">
                        <span class="toggle-label">{autoPaste ? 'Auto Paste' : 'Not Paste'}</span>
                    </span>
                    <span class="toggle-indicator">
                        <span class="toggle-dot"></span>
                    </span>
                </button>
                
                <div class="search-wrapper">
                    <input 
                        type="search" 
                        class="search-input" 
                        placeholder="Search keys..." 
                        bind:value={searchQuery}
                        on:keydown={handleSearchKeydown}
                    >
                </div>

                <div class="action-wrapper">
                    <button class="icon-btn add-btn" on:click={toggleMenu} title="New Item">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="12" y1="5" x2="12" y2="19"></line>
                            <line x1="5" y1="12" x2="19" y2="12"></line>
                        </svg>
                    </button>
                    {#if showMenu}
                        <div class="dropdown-menu" on:click|stopPropagation on:keydown|stopPropagation in:fly={{ y: -5, duration: 150, easing: iosElastic }} out:fade={{duration: 70}}>
                            <button on:click={addText}>文本 (Text)</button>
                            <button on:click={addDir}>文件夹 (Folder)</button>
                            <div class="dropdown-divider"></div>
                            <button on:click={openSettings}>偏好设置 (Settings)</button>
                        </div>
                    {/if}
                </div>
            </div>

            {#if hasActiveSlots && !searchQuery.trim()}
                <div class="suggestion-bar-vertical" transition:slide={{ duration: 160, easing: cubicOut }}>
                    <div class="suggestion-tag">
                        <span>常</span>
                        <span>用</span>
                    </div>
                    <div class="suggestion-slot-list">
                        {#each capsuleSlots as slot, idx}
                            {#if slot.type === 'pinned' && slot.item}
                                <button 
                                    class="slot-item pinned"
                                    on:click={() => handleSlotClick(slot)}
                                    on:contextmenu={(e) => handleCapsuleContextMenu(e, slot)}
                                    title={slot.item.value || ''}
                                >
                                    <svg class="slot-pin-svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                                        <line x1="12" y1="17" x2="12" y2="22"></line>
                                        <path d="M5 17h14v-2l-2-2V5h1V3H6v2h1v8l-2 2v2z"></path>
                                    </svg>
                                    <span class="slot-title">{slot.item.name}</span>
                                    {#if activeSlotsCount === 1}
                                        <span class="slot-badge enter-badge">
                                            <svg class="enter-icon" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                                                <polyline points="9 10 4 15 9 20"></polyline>
                                                <path d="M20 4v7a4 4 0 0 1-4 4H4"></path>
                                            </svg>
                                            <span>Enter</span>
                                        </span>
                                    {:else if getCapsuleBadgeLabel(idx)}
                                        <span class="slot-badge {isEnterKey(capsuleShortcuts[idx]) ? 'enter-badge' : 'shortcut-badge'}">
                                            {#if isEnterKey(capsuleShortcuts[idx])}
                                                <svg class="enter-icon" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                                                    <polyline points="9 10 4 15 9 20"></polyline>
                                                    <path d="M20 4v7a4 4 0 0 1-4 4H4"></path>
                                                </svg>
                                            {/if}
                                            <span>{getCapsuleBadgeLabel(idx)}</span>
                                        </span>
                                    {/if}
                                </button>
                            {:else if slot.type === 'auto' && slot.item}
                                <button 
                                    class="slot-item auto"
                                    on:click={() => handleSlotClick(slot)}
                                    on:contextmenu={(e) => handleCapsuleContextMenu(e, slot)}
                                    title={slot.item.value || ''}
                                >
                                    <!-- 自动频次不加符号 -->
                                    <span class="slot-title">{slot.item.name}</span>
                                    {#if activeSlotsCount === 1}
                                        <span class="slot-badge enter-badge">
                                            <svg class="enter-icon" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                                                <polyline points="9 10 4 15 9 20"></polyline>
                                                <path d="M20 4v7a4 4 0 0 1-4 4H4"></path>
                                            </svg>
                                            <span>Enter</span>
                                        </span>
                                    {:else if getCapsuleBadgeLabel(idx)}
                                        <span class="slot-badge {isEnterKey(capsuleShortcuts[idx]) ? 'enter-badge' : 'shortcut-badge'}">
                                            {#if isEnterKey(capsuleShortcuts[idx])}
                                                <svg class="enter-icon" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                                                    <polyline points="9 10 4 15 9 20"></polyline>
                                                    <path d="M20 4v7a4 4 0 0 1-4 4H4"></path>
                                                </svg>
                                            {/if}
                                            <span>{getCapsuleBadgeLabel(idx)}</span>
                                        </span>
                                    {/if}
                                </button>
                            {:else}
                                <div class="slot-item empty" title="右键下方列表条目可固定至此槽位">
                                    <svg class="slot-empty-svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <line x1="12" y1="5" x2="12" y2="19"></line>
                                        <line x1="5" y1="12" x2="19" y2="12"></line>
                                    </svg>
                                    <span class="slot-title empty-label">待分配</span>
                                    {#if getCapsuleBadgeLabel(idx)}
                                        <span class="slot-badge empty-badge">
                                            <span>{getCapsuleBadgeLabel(idx)}</span>
                                        </span>
                                    {/if}
                                </div>
                            {/if}
                        {/each}
                    </div>
                </div>
            {/if}
        </div>
        
        <div class="content-scrollable">
            {#if searchQuery.trim()}
                <div class="search-results-overlay">
                    {#if searchResults.length > 0}
                        {#each searchResults as result}
                            <div class="search-result-item" 
                                on:click={() => handleSearchResultClick(result)}
                                on:keydown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleSearchResultClick(result);
                                    }
                                }}
                            >
                                <div class="result-path">{result.fullPath}</div>
                                <div class="result-name">{result.name}</div>
                            </div>
                        {/each}
                    {:else}
                        <div class="no-results">No matches found</div>
                    {/if}
                </div>
            {:else}
                {#if data.length === 0}
                    <div class="empty-state">No Items</div>
                {:else}
                    <ul class="tree-root">
                        {#each data as node (node.id)}
                            <TreeItem 
                                {node}
                                {expanded} 
                                {toggleExpand} 
                                {showContextMenu}
                                onMoveNode={handleMoveNode}
                                {autoPaste}
                            />
                        {/each}
                    </ul>
                {/if}
            {/if}
        </div>
    {/if}
</div>

<ContextMenu 
    visible={globalContextMenu.visible}
    x={globalContextMenu.x}
    y={globalContextMenu.y}
    flipX={globalContextMenu.flipX}
    flipY={globalContextMenu.flipY}
    targetNode={globalContextMenu.targetNode}
    isCapsule={globalContextMenu.isCapsule}
    capsuleSlot={globalContextMenu.capsuleSlot}
    slots={capsuleSlots}
    capsuleShortcuts={capsuleShortcuts}
    on:addText={addText}
    on:addDir={addDir}
    on:editText={editText}
    on:editDir={editDir}
    on:removeCapsule={removeCapsule}
    on:delete={deleteItem}
    on:assignSlot={handleAssignSlot}
    on:unpinSlot={handleUnpinSlot}
    on:pinHere={handlePinHere}
/>

<DirModal 
    visible={showDirInput}
    isEdit={isEditDirMode}
    initialName={dirModalName}
    on:submit={handleDirSubmit}
    on:cancel={closeDirModal}
/>

<TextModal 
    visible={showTextInput}
    isEdit={isEditTextMode}
    initialTitle={textModalTitle}
    initialValue={textModalValue}
    on:submit={handleTextSubmit}
    on:cancel={closeTextModal}
/>

<ConfirmModal 
    visible={showDeleteConfirm}
    title={deleteType === 'capsule' ? '移除常用推荐' : '确认删除'}
    message={itemToDelete ? (deleteType === 'capsule' ? `确定从常用推荐中移除 "${itemToDelete.name}" 吗？` : `确定要彻底删除 "${itemToDelete.name}" 吗？此操作不可撤销。`) : ''}
    confirmText={deleteType === 'capsule' ? '移除' : '删除'}
    cancelText="取消"
    on:confirm={confirmDeleteItem}
    on:cancel={cancelDelete}
/>

<ConfirmModal 
    visible={showSlotConfirm}
    title="替换常用槽位"
    message={pendingSlotTarget.newNode && pendingSlotTarget.currentItem ? `槽位 ${pendingSlotTarget.slot} 当前为 "${pendingSlotTarget.currentItem.name}"，确定替换为 "${pendingSlotTarget.newNode.name}" 吗？` : ''}
    confirmText="确认替换"
    cancelText="取消"
    confirmType="primary"
    on:confirm={confirmSlotReplace}
    on:cancel={cancelSlotReplace}
/>

<style>
    .app-container {
        width: 100%;
        height: 100%;
        background: rgba(248, 250, 252, 0.68);
        -webkit-backdrop-filter: blur(28px) saturate(1.28);
        backdrop-filter: blur(28px) saturate(1.28);
        border: 1px solid rgba(255, 255, 255, 0.68);
        border-radius: 8px;
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.72),
            inset 0 0 0 1px rgba(99, 119, 139, 0.06),
            0 8px 24px rgba(25, 38, 52, 0.16);
        display: flex;
        flex-direction: column;
        overflow: hidden;
        isolation: isolate;
    }

    .sticky-header {
        flex-shrink: 0;
        background: rgba(244, 248, 251, 0.52);
        -webkit-backdrop-filter: blur(24px) saturate(1.35);
        backdrop-filter: blur(24px) saturate(1.35);
        border-bottom: 1px solid rgba(74, 96, 116, 0.1);
        box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.62);
        padding: 8px 10px 4px;
        z-index: 10;
    }

    .header-row {
        display: flex;
        align-items: center;
        gap: 8px;
        height: 28px;
    }

    .suggestion-bar-vertical {
        display: flex;
        align-items: stretch;
        gap: 6px;
        margin-top: 6px;
        padding-top: 5px;
        border-top: 1px solid rgba(71, 85, 105, 0.09);
    }

    .suggestion-tag {
        flex-shrink: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 6px;
        width: 22px;
        box-sizing: border-box;
        font-size: 11px;
        font-weight: 600;
        color: #475569;
        background: rgba(71, 85, 105, 0.06);
        border: 1px solid rgba(71, 85, 105, 0.12);
        border-radius: 6px;
        user-select: none;
        padding: 4px 0;
        box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.6);
    }

    .suggestion-tag span {
        line-height: 1;
        display: block;
    }

    .suggestion-slot-list {
        display: flex;
        flex-direction: column;
        gap: 3px;
        flex: 1;
        min-width: 0;
    }

    .slot-item {
        display: flex;
        align-items: center;
        gap: 6px;
        height: var(--app-chip-height, 24px);
        padding: 0 8px;
        border-radius: 6px;
        font-size: var(--app-font-size, 13px);
        width: 100%;
        box-sizing: border-box;
        text-align: left;
        user-select: none;
        transition: all 0.18s cubic-bezier(0.34, 1.4, 0.64, 1);
    }

    /* 手动固定项 (Pinned): 纯净白底、实体高光微光、细线边框、图钉图标 */
    .slot-item.pinned {
        background: rgba(255, 255, 255, 0.92);
        border: 1px solid rgba(100, 135, 175, 0.42);
        color: #0f172a;
        font-weight: 550;
        cursor: pointer;
        box-shadow:
            inset 0 1px 0 #ffffff,
            0 1px 3px rgba(15, 23, 42, 0.05);
    }

    .slot-item.pinned:hover {
        background: #ffffff;
        border-color: rgba(70, 115, 165, 0.65);
        box-shadow:
            inset 0 1px 0 #ffffff,
            0 2px 7px rgba(15, 23, 42, 0.08);
    }

    .slot-pin-svg {
        flex-shrink: 0;
        color: #2563eb;
        opacity: 0.95;
    }

    /* 自动高频项 (Auto): 半透明磨砂、无图标、标准字重 */
    .slot-item.auto {
        background: rgba(255, 255, 255, 0.64);
        border: 1px solid rgba(148, 163, 184, 0.28);
        color: #1e293b;
        font-weight: 500;
        cursor: pointer;
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.9),
            0 1px 2px rgba(15, 23, 42, 0.03);
    }

    .slot-item.auto:hover {
        background: rgba(255, 255, 255, 0.92);
        border-color: rgba(100, 116, 139, 0.38);
        color: #0f172a;
        box-shadow:
            inset 0 1px 0 #ffffff,
            0 2px 6px rgba(15, 23, 42, 0.06);
    }

    /* 空槽 (Empty): 浅灰虚线边框、半透明文字、加号提示 */
    .slot-item.empty {
        background: rgba(255, 255, 255, 0.16);
        border: 1px dashed rgba(148, 163, 184, 0.38);
        color: #94a3b8;
        cursor: default;
    }

    .slot-empty-svg {
        flex-shrink: 0;
        color: #94a3b8;
        opacity: 0.75;
    }

    .empty-label {
        font-style: italic;
        color: #94a3b8 !important;
        font-weight: 400 !important;
    }

    .slot-title {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        flex: 1 1 auto;
        min-width: 0;
        line-height: var(--app-chip-height, 24px);
    }

    /* 快捷键徽标 */
    .slot-badge {
        flex-shrink: 0;
        display: inline-flex;
        align-items: center;
        gap: 3.5px;
        font-size: 11px;
        font-weight: 600;
        line-height: 1;
        border-radius: 4px;
        letter-spacing: 0.2px;
        user-select: none;
        pointer-events: none;
        white-space: nowrap;
        padding: 2.5px 6.5px;
        margin-left: 6px;
    }

    .slot-badge.shortcut-badge {
        border: 1px solid rgba(140, 170, 200, 0.45);
        border-bottom: 1.5px solid rgba(110, 145, 180, 0.68);
        background: #ffffff;
        color: #1a4266;
        box-shadow:
            0 1px 2px rgba(25, 45, 70, 0.06),
            inset 0 1px 0 #ffffff;
    }

    .slot-item.pinned .slot-badge.shortcut-badge {
        border-color: rgba(37, 99, 235, 0.42);
        border-bottom-color: rgba(29, 78, 216, 0.7);
        color: #1d4ed8;
    }

    .slot-badge.enter-badge {
        border: 1px solid rgba(140, 170, 200, 0.45);
        border-bottom: 1.5px solid rgba(110, 145, 180, 0.68);
        background: #ffffff;
        color: #1a4266;
        box-shadow:
            0 1px 2px rgba(25, 45, 70, 0.06),
            inset 0 1px 0 #ffffff;
    }

    .slot-badge.empty-badge {
        border: 1px dashed rgba(148, 163, 184, 0.32);
        background: transparent;
        color: #94a3b8;
        opacity: 0.75;
    }

    .slot-badge .enter-icon {
        opacity: 0.8;
        flex-shrink: 0;
    }

    .search-wrapper {
        flex: 1;
        position: relative;
    }

    .search-input {
        width: 100%;
        height: 26px;
        border: 1px solid rgba(255, 255, 255, 0.72);
        background: rgba(255, 255, 255, 0.46);
        border-radius: 6px;
        padding: 0 9px;
        color: #263442;
        font-family: inherit;
        font-size: var(--app-font-size, 13px);
        outline: none;
        box-shadow:
            inset 0 0 0 1px rgba(75, 98, 119, 0.07),
            0 1px 3px rgba(32, 48, 63, 0.06);
        transition: background-color 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
    }

    .search-input::placeholder {
        color: rgba(62, 78, 94, 0.55);
    }

    .search-input:focus {
        background: rgba(255, 255, 255, 0.72);
        border-color: rgba(111, 183, 220, 0.58);
        box-shadow:
            inset 0 0 0 1px rgba(255, 255, 255, 0.55),
            0 0 0 2px rgba(73, 156, 201, 0.13),
            0 3px 10px rgba(35, 81, 108, 0.08);
    }

    .action-wrapper {
        position: relative;
    }

    .icon-btn {
        background: rgba(255, 255, 255, 0.22);
        border: 1px solid rgba(255, 255, 255, 0.5);
        border-radius: 6px;
        width: 26px;
        height: 26px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #536475;
        cursor: pointer;
        box-shadow: inset 0 0 0 1px rgba(74, 97, 117, 0.04);
        transition: background-color 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease, color 0.18s ease;
        padding: 0;
    }

    .icon-btn:hover {
        background: rgba(226, 242, 250, 0.72);
        border-color: rgba(255, 255, 255, 0.82);
        color: #236b96;
        box-shadow:
            inset 0 0 0 1px rgba(77, 151, 190, 0.12),
            0 2px 8px rgba(31, 80, 107, 0.1);
    }

    .icon-btn svg {
        opacity: 0.8;
    }

    /* --- 粘贴模式切换开关 (iOS 弹性收缩展开) --- */
    .paste-toggle {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 6px 2px 4px;
        border: 1px solid rgba(255, 255, 255, 0.56);
        border-radius: 20px;
        background: rgba(255, 255, 255, 0.28);
        box-shadow: inset 0 0 0 1px rgba(71, 94, 115, 0.045);
        cursor: pointer;
        user-select: none;
        outline: none;
        font-family: inherit;
        font-size: 13px;
        white-space: nowrap;
        margin-right: 4px;
        flex-shrink: 0;
        transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease, gap 0.28s ease, padding 0.28s ease;
        overflow: hidden;
    }

    .paste-toggle:hover {
        gap: 8px;
        padding: 2px 8px 2px 10px;
        background: rgba(255, 255, 255, 0.65);
        border-color: rgba(255, 255, 255, 0.86);
        box-shadow:
            inset 0 0 0 1px rgba(72, 139, 177, 0.09),
            0 2px 8px rgba(29, 68, 91, 0.08);
    }

    .paste-toggle:focus-visible {
        box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
        transition: all 0.5s cubic-bezier(0.34, 1.56, 0.64, 1), box-shadow 0.25s cubic-bezier(0.34, 1.3, 0.64, 1);
    }

    .toggle-text {
        overflow: hidden;
        max-width: 20px;
        transition: max-width 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .paste-toggle:hover .toggle-text {
        max-width: 100px;
    }

    .toggle-label {
        display: inline-block;
        font-weight: 500;
        font-size: 12px;
        color: #666;
        white-space: nowrap;
        transition: color 0.35s cubic-bezier(0.34, 1.3, 0.64, 1);
    }

    .paste-toggle.active .toggle-label {
        color: #2563eb;
    }

    .toggle-indicator {
        position: relative;
        width: 28px;
        height: 14px;
        background: #d1d5db;
        border-radius: 14px;
        transition: background 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
        flex-shrink: 0;
    }

    .paste-toggle.active .toggle-indicator {
        background: #93c5fd;
    }

    .toggle-dot {
        position: absolute;
        top: 2px;
        left: 2px;
        width: 10px;
        height: 10px;
        background: #fff;
        border-radius: 50%;
        box-shadow: 0 1px 2px rgba(0,0,0,0.15);
        transition: transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .paste-toggle.active .toggle-dot {
        transform: translateX(14px);
        background: #3b82f6;
        box-shadow: 0 1px 3px rgba(59, 130, 246, 0.3);
    }

    .dropdown-menu {
        position: absolute;
        top: 30px;
        right: 0;
        background: rgba(248, 250, 252, 0.82);
        -webkit-backdrop-filter: blur(22px) saturate(1.3);
        backdrop-filter: blur(22px) saturate(1.3);
        border: 1px solid rgba(255, 255, 255, 0.72);
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.65),
            0 8px 22px rgba(27, 43, 57, 0.17);
        border-radius: 7px;
        padding: 4px;
        min-width: 120px;
        z-index: 9999;
    }

    .dropdown-menu button {
        display: block;
        width: 100%;
        text-align: left;
        background: none;
        border: none;
        padding: 6px 12px;
        font-size: 13px;
        color: #333;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .dropdown-menu button:hover {
        background: rgba(213, 235, 247, 0.76);
        color: #215f82;
        box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.58);
    }

    .dropdown-divider {
        height: 1px;
        background: rgba(74, 96, 116, 0.12);
        margin: 4px 2px;
    }

    .content-scrollable {
        flex: 1;
        min-width: 0;
        overflow-x: hidden;
        overflow-y: auto;
        padding: 4px 0;
    }

    .content-scrollable::-webkit-scrollbar {
        width: 2px;
    }

    .content-scrollable::-webkit-scrollbar-track {
        background: transparent;
    }

    .content-scrollable::-webkit-scrollbar-thumb {
        background: rgba(30, 41, 59, 0.08);
        border-radius: 2px;
        transition: background-color 0.2s ease;
    }

    .content-scrollable:hover::-webkit-scrollbar-thumb {
        background: rgba(30, 41, 59, 0.16);
    }

    .content-scrollable::-webkit-scrollbar-thumb:hover {
        background: rgba(30, 41, 59, 0.25);
    }

    .empty-state {
        text-align: center;
        color: #999;
        margin-top: 40px;
        font-size: 13px;
    }

    .tree-root {
        list-style: none;
        padding: 0;
        margin: 0;
    }



    .search-results-overlay {
        background: rgba(255, 255, 255, 0.16);
        min-height: 100%;
        padding-top: 5px;
    }

    .search-result-item {
        padding: 8px 15px;
        border-bottom: 1px solid rgba(0,0,0,0.03);
        cursor: pointer;
        transition: all 0.25s cubic-bezier(0.34, 1.3, 0.64, 1);
    }

    .search-result-item:hover {
        background: rgba(224, 239, 248, 0.58);
        box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.5);
    }

    .result-path {
        font-size: 10px;
        color: #999;
        margin-bottom: 2px;
    }

    .result-name {
        font-size: 13px;
        font-weight: 500;
        color: #333;
    }

    .no-results {
        padding: 20px;
        text-align: center;
        color: #999;
        font-size: 13px;
    }
</style>
