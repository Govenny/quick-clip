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
    import { EnterSettingsMode, GetConfig, GetContent, SaveContent, ExitSettingsMode, ToggleWindow, HideWindow, GetContextSuggestions, RecordItemUsage} from '../wailsjs/go/main/App'; 
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
    let suggestedItems = [];
    let hoveredExpandedIdx = null;
    let truncatedMap = {};

    async function updateTruncationStatus() {
        await tick();
        const chips = document.querySelectorAll('.suggestion-chips .suggestion-chip');
        const newMap = {};
        chips.forEach((chip, idx) => {
            const titleEl = chip.querySelector('.chip-title');
            if (titleEl && titleEl.scrollWidth > titleEl.clientWidth + 1) {
                newMap[idx] = true;
            }
        });
        truncatedMap = newMap;
    }

    // 粘贴模式开关: true=Auto Paste, false=Not Paste
    let autoPaste = true;

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

    // 删除确认弹窗
    let showDeleteConfirm = false;
    let itemToDelete = null;

    // 全局上下文菜单
    let globalContextMenu = {
        visible: false,
        x: 0,
        y: 0,
        targetNode: null,
        flipX: false,
        flipY: false
    };

    function cleanGlobalContextMenu() {
        globalContextMenu = {
            visible: false,
            x: 0,
            y: 0,
            targetNode: null,
            flipX: false,
            flipY: false
        };
    }

    function showContextMenu(e, node) {
        e.preventDefault();
        e.stopPropagation();
        
        const isFolder = node && node.type === 'folder';
        const menuWidth = 128;
        const itemHeight = 28;
        const dividerHeight = 9;
        const padding = 8;
        
        let itemCount, dividerCount;
        if (isFolder) {
            itemCount = 4; // New Text + New Folder + Edit + Delete
            dividerCount = 2;
        } else {
            itemCount = 2; // Edit + Delete
            dividerCount = 1;
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

    function deleteItem() {
        if (!globalContextMenu.targetNode) return;
        itemToDelete = globalContextMenu.targetNode;
        showDeleteConfirm = true;
        hideContextMenu();
    }

    function confirmDeleteItem() {
        if (!itemToDelete) return;

        try {
            deleteNodeById(data, itemToDelete.id);
            updateData([...data]);
            cancelDelete();
        } catch (err) {
            console.error("删除失败", err);
        }
    }

    function cancelDelete() {
        showDeleteConfirm = false;
        itemToDelete = null;
        cleanGlobalContextMenu();
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
        await tick();
        updateTruncationStatus();
    }

    // 监听来自后端的 update-content 事件
    const contentEventListener = async () => {
        try {
            LogInfo("update-content发送成功");
            const newData = await GetContent();
            data = normalizeTree(newData);
            await tick();
        } catch (error) {
            console.error('Failed to load content:', error);
        }
    };

    async function loadSuggestions() {
        hoveredExpandedIdx = null;
        try {
            const topIds = await GetContextSuggestions();
            if (topIds && topIds.length > 0) {
                const found = [];
                for (const id of topIds) {
                    const node = findNodeById(data, id);
                    if (node && node.type === 'text') {
                        found.push(node);
                    }
                }
                suggestedItems = found;
                updateTruncationStatus();
            } else {
                suggestedItems = [];
                truncatedMap = {};
            }
        } catch (err) {
            suggestedItems = [];
            truncatedMap = {};
        }
    }

    onMount(() => {
        document.addEventListener('click', handleGlobalClick);
        document.addEventListener('contextmenu', hideContextMenu);

        EventsOn("show-settings", settingsEventListener);
        EventsOn("update-content", contentEventListener);
        EventsOn("window-shown", async () => {
            await loadSuggestions();
        });
    });

    onMount(async () => {
        try {
            const cfg = await GetConfig();
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
        hoveredExpandedIdx = null;
        setTimeout(() => {
            if (document.hasFocus()) {
                return;
            }

            if (showTextInput || showDirInput || showSettings) {
                return; 
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

    function handleChipMouseEnter(event, idx) {
        let isTruncated = truncatedMap[idx];
        if (isTruncated === undefined) {
            const btn = event.currentTarget;
            const titleEl = btn.querySelector('.chip-title');
            isTruncated = titleEl && titleEl.scrollWidth > titleEl.clientWidth + 1;
        }

        // 仅当文字在默认等分紧凑宽度下显示不完整（被截断）时才触发展开与压缩
        if (isTruncated) {
            hoveredExpandedIdx = idx;
        } else {
            hoveredExpandedIdx = null;
        }
    }

    function handleChipMouseLeave() {
        hoveredExpandedIdx = null;
    }

    function handleSuggestionClick(item) {
        hoveredExpandedIdx = null;
        if (!item) return;
        if (item.id) {
            RecordItemUsage(item.id);
        }
        const content = item.value || '';
        navigator.clipboard.writeText(content).then(() => {
            if (autoPaste) {
                PasteAndHide();
            } else {
                HideAndRestore();
            }
        }).catch(err => console.error("Suggestion copy failed:", err));
    }

    function handleSearchResultClick(result) {
        if (result && result.id) {
            RecordItemUsage(result.id);
        }
        const content = typeof result === 'string' ? result : (result.content || '');
        navigator.clipboard.writeText(content).then(() => {
            if (autoPaste) {
                PasteAndHide();
            } else {
                HideAndRestore();
            }
            searchQuery = "";
        }).catch(err => console.error("Search copy failed:", err));
    }

    function handleGlobalKeydown(e) {
        if (e.key === 'Enter') {
            if (showTextInput || showDirInput || showDeleteConfirm || showSettings) {
                return;
            }
            if (document.activeElement && document.activeElement.classList.contains('paste-toggle')) {
                return;
            }
            if (searchQuery.trim()) {
                return;
            }
            // 仅当且仅当恰好只有 1 个常用胶囊时，敲回车直接点击胶囊
            if (suggestedItems && suggestedItems.length === 1) {
                e.preventDefault();
                handleSuggestionClick(suggestedItems[0]);
            }
        }
    }

    function handleSearchKeydown(e) {
        if (e.key === 'Enter') {
            if (!searchQuery.trim() && suggestedItems && suggestedItems.length === 1) {
                e.preventDefault();
                handleSuggestionClick(suggestedItems[0]);
            } else if (searchQuery.trim() && searchResults.length > 0) {
                e.preventDefault();
                handleSearchResultClick(searchResults[0]);
            }
        }
    }
</script>

<svelte:window 
    on:blur={() => handleBlur()} 
    on:resize={() => updateTruncationStatus()}
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

            {#if suggestedItems && suggestedItems.length > 0 && !searchQuery.trim()}
                <div class="suggestion-bar" transition:slide={{ duration: 160, easing: cubicOut }}>
                    <span class="suggestion-tag">常用</span>
                    <div class="suggestion-chips" class:has-expanded={hoveredExpandedIdx !== null}>
                        {#each suggestedItems as item, idx}
                            <button 
                                class="suggestion-chip" 
                                class:hover-expand={hoveredExpandedIdx === idx}
                                on:mouseenter={(e) => handleChipMouseEnter(e, idx)}
                                on:mouseleave={handleChipMouseLeave}
                                on:focus={(e) => handleChipMouseEnter(e, idx)}
                                on:blur={handleChipMouseLeave}
                                on:click={() => handleSuggestionClick(item)}
                                title={item.value || ''}
                            >
                                <svg class="chip-icon" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                                    <polyline points="14 2 14 8 20 8"></polyline>
                                    <line x1="16" y1="13" x2="8" y2="13"></line>
                                    <line x1="16" y1="17" x2="8" y2="17"></line>
                                </svg>
                                <span class="chip-title">{item.name}</span>
                                {#if suggestedItems.length === 1}
                                    <span class="chip-badge">
                                        <svg class="enter-icon" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                                            <polyline points="9 10 4 15 9 20"></polyline>
                                            <path d="M20 4v7a4 4 0 0 1-4 4H4"></path>
                                        </svg>
                                        <span>Enter</span>
                                    </span>
                                {/if}
                            </button>
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
                                on:click={() => handleSearchResultClick(result.content)}
                                on:keydown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleSearchResultClick(result.content);
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
    on:addText={addText}
    on:addDir={addDir}
    on:editText={editText}
    on:editDir={editDir}
    on:delete={deleteItem}
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
    title="Confirm Delete"
    message={itemToDelete ? `Are you sure you want to delete "${itemToDelete.name}"?` : ''}
    confirmText="Delete"
    cancelText="Cancel"
    on:confirm={confirmDeleteItem}
    on:cancel={cancelDelete}
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

    .suggestion-bar {
        display: flex;
        align-items: center;
        gap: 7px;
        margin-top: 6px;
        padding-top: 5px;
        border-top: 1px solid rgba(71, 85, 105, 0.09);
        overflow: hidden;
    }

    .suggestion-tag {
        flex-shrink: 0;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        height: 20px;
        padding: 0 6px;
        font-size: 11px;
        font-weight: 600;
        color: #475569;
        background: rgba(71, 85, 105, 0.07);
        border: 1px solid rgba(71, 85, 105, 0.12);
        border-radius: 5px;
        letter-spacing: 0.3px;
        user-select: none;
    }

    .suggestion-chips {
        display: flex;
        align-items: center;
        gap: 5px;
        overflow: hidden;
        flex: 1;
        min-width: 0;
        height: var(--app-chip-height, 24px);
    }

    .suggestion-chip {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        padding: 0 8px;
        background: rgba(255, 255, 255, 0.72);
        border: 1px solid rgba(148, 163, 184, 0.28);
        border-radius: 12px;
        font-size: var(--app-font-size, 13px);
        font-weight: 550;
        color: #1e293b;
        cursor: pointer;
        height: var(--app-chip-height, 24px);
        box-sizing: border-box;
        white-space: nowrap;
        overflow: hidden;
        /* 默认等分占比 1:1:1 */
        flex: 1 1 0;
        min-width: 28px;
        /* 拟物磨砂质感：高光顶缘 + 板岩微柔光阴影，不靠明艳高饱和颜色，自然浮现立体层次 */
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.95),
            0 1px 2px rgba(15, 23, 42, 0.04),
            0 2px 4px rgba(15, 23, 42, 0.02);
        /* iOS 弹簧贝塞尔曲线 (与 Auto Paste 保持一致) */
        transition: flex 0.38s cubic-bezier(0.34, 1.4, 0.64, 1),
                    background-color 0.2s ease,
                    border-color 0.2s ease,
                    box-shadow 0.2s ease,
                    color 0.2s ease,
                    padding 0.28s ease,
                    opacity 0.2s ease;
    }

    .chip-icon {
        flex-shrink: 0;
        color: #64748b;
        opacity: 0.85;
        transition: color 0.18s ease, opacity 0.18s ease;
    }

    .suggestion-chip:hover {
        background: rgba(255, 255, 255, 0.94);
        border-color: rgba(100, 116, 139, 0.36);
        color: #0f172a;
        box-shadow:
            inset 0 1px 0 #ffffff,
            0 2px 6px rgba(15, 23, 42, 0.07),
            0 0 0 1px rgba(148, 163, 184, 0.1);
    }

    .suggestion-chip:hover .chip-icon {
        color: #334155;
        opacity: 1;
    }

    /* 当存在展开项时，压缩未悬停的兄弟项 */
    .suggestion-chips.has-expanded .suggestion-chip:not(.hover-expand) {
        flex: 0.5 1 0;
        padding: 0 7px;
        opacity: 0.65;
        background: rgba(255, 255, 255, 0.45);
        border-color: rgba(148, 163, 184, 0.2);
    }

    /* 截断项悬停展开：纯净白玉浮雕微光 */
    .suggestion-chip.hover-expand {
        flex: 3.5 1 0;
        background: #ffffff;
        border-color: rgba(71, 85, 105, 0.42);
        color: #0f172a;
        box-shadow:
            inset 0 1px 0 #ffffff,
            0 3px 10px rgba(15, 23, 42, 0.09),
            0 0 0 1px rgba(71, 85, 105, 0.12);
    }

    .suggestion-chip.hover-expand .chip-icon {
        color: #1e293b;
        opacity: 1;
    }

    .chip-title {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        flex: 1 1 auto;
        min-width: 0;
        text-align: left;
        line-height: var(--app-chip-height, 24px);
    }

    .chip-badge {
        flex-shrink: 0;
        display: inline-flex;
        align-items: center;
        gap: 3.5px;
        font-size: 11px;
        font-weight: 600;
        line-height: 1;
        padding: 2.5px 6.5px;
        margin-left: 6px;
        color: #2b5074;
        background: rgba(240, 246, 252, 0.94);
        border: 1px solid rgba(140, 170, 200, 0.42);
        border-bottom: 1.5px solid rgba(105, 140, 175, 0.65);
        border-radius: 4px;
        box-shadow:
            0 1px 2px rgba(15, 23, 42, 0.05),
            inset 0 1px 0 rgba(255, 255, 255, 0.95);
        letter-spacing: 0.2px;
        user-select: none;
        transition: all 0.18s ease;
    }

    .chip-badge .enter-icon {
        opacity: 0.75;
        flex-shrink: 0;
        transition: opacity 0.18s ease, transform 0.18s ease;
    }

    .suggestion-chip:hover .chip-badge {
        background: #ffffff;
        color: #1a4266;
        border-color: rgba(90, 130, 168, 0.55);
        border-bottom-color: rgba(70, 115, 155, 0.8);
        box-shadow:
            0 2px 5px rgba(25, 45, 70, 0.09),
            inset 0 1px 0 #ffffff;
    }

    .suggestion-chip:hover .chip-badge .enter-icon {
        opacity: 1;
        transform: translateX(-1px);
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
