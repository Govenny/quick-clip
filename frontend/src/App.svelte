<script>
    import { onMount, tick, onDestroy } from 'svelte';
    import { fade, fly, scale, slide } from 'svelte/transition';

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
    import { quartOut, cubicOut } from 'svelte/easing';
    import { EnterSettingsMode, GetContent, SaveContent, ExitSettingsMode, ToggleWindow, HideWindow, GetContextSuggestions, RecordItemUsage} from '../wailsjs/go/main/App'; 
    import { LogInfo, EventsOn } from '../wailsjs/runtime';
    import TreeItem from './components/TreeItem.svelte';
    import Setting from './components/Setting.svelte';
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

    // 编辑模式
    let isEditMode = false;
    let editingNode = null;
    let targetFolderNode = null;

    // 添加目录
    let showDirInput = false;
    let dirName = "";
    let dirInputRef; 

    // 添加文本
    let showTextInput = false;
    let titleName = "";
    let titleInputRef;
    let textName = "";
    let textInputRef;

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
            const rawData = await GetContent();
            data = normalizeTree(rawData);
            await loadSuggestions();
        } catch (error) {
            console.error('Failed to load content:', error);
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
        isEditMode = false;
        editingNode = null;
        targetFolderNode = (globalContextMenu.targetNode && globalContextMenu.targetNode.type === 'folder')
            ? globalContextMenu.targetNode
            : null;

        showTextInput = true;
        titleName = "";
        textName = "";
        showMenu = false;
        hideContextMenu();

        tick().then(() => {
            if (titleInputRef) {
                titleInputRef.focus();
            }
        });
    }

    // 表单验证：名称与内容不能为空，点号已完全允许
    $: isFormValid = titleName.trim() !== "" && textName.trim() !== "";

    function handleKeyDown(event, isTitleInput) {
        const { key } = event;

        if (key === 'Escape') {
            cancelAddText();
            return;
        }

        if (key === 'Enter') {
            if (isTitleInput) {
                event.preventDefault();
                textInputRef?.focus();
                return;
            }

            // 内容框使用 Shift+Enter 换行；Enter 保存
            if (event.shiftKey) {
                return;
            }

            event.preventDefault();
            if (isFormValid) {
                confirmAddText();
            }
        } else if (key === 'Tab' && isTitleInput && event.shiftKey === false) {
            event.preventDefault();
            textInputRef?.focus();
        }
    }

    function confirmAddText() {
        const trimmedTitle = titleName.trim();
        if (!trimmedTitle || !textName) {
            alert("请完善输入");
            return;
        }

        if (isEditMode && editingNode) {
            editingNode.name = trimmedTitle;
            editingNode.value = textName;
            updateData([...data]);
        } else {
            const newNode = {
                id: generateId(),
                name: trimmedTitle,
                type: 'text',
                value: textName
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

        cancelAddText();
    }

    function cancelAddText() {
        titleName = "";
        textName = "";
        showTextInput = false;
        isEditMode = false;
        editingNode = null;
        targetFolderNode = null;
        cleanGlobalContextMenu();
    }

    function editText() {
        if (!globalContextMenu.targetNode) return;
        isEditMode = true;
        editingNode = globalContextMenu.targetNode;
        targetFolderNode = null;
        showTextInput = true;
        titleName = editingNode.name;
        textName = editingNode.value || "";
        
        showMenu = false;
        hideContextMenu();

        tick().then(() => titleInputRef?.focus());
    }

    $: if (showTextInput && titleInputRef) {
        setTimeout(() => titleInputRef.focus(), 0);
    }

    function addDir() {
        isEditMode = false;
        editingNode = null;
        targetFolderNode = (globalContextMenu.targetNode && globalContextMenu.targetNode.type === 'folder')
            ? globalContextMenu.targetNode
            : null;

        showDirInput = true;
        dirName = "";
        showMenu = false;
        hideContextMenu();

        tick().then(() => {
            if (dirInputRef) {
                dirInputRef.focus();
            }
        });
    }

    function confirmAddDir() {
        const newDirName = dirName.trim();
        if (!newDirName) {
            alert("名称无效");
            return;
        }

        if (isEditMode && editingNode) {
            editingNode.name = newDirName;
            updateData([...data]);
        } else {
            const newNode = {
                id: generateId(),
                name: newDirName,
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

        cancelAddDir();
    }

    function cancelAddDir() {
        showDirInput = false;
        dirName = "";
        isEditMode = false;
        editingNode = null;
        targetFolderNode = null;
        cleanGlobalContextMenu();
    }

    function editDir() {
        if (!globalContextMenu.targetNode) return;
        isEditMode = true;
        editingNode = globalContextMenu.targetNode;
        targetFolderNode = null;
        showDirInput = true;
        dirName = editingNode.name;
        
        showMenu = false;
        hideContextMenu();

        tick().then(() => dirInputRef?.focus());
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

    function handleSearchKeydown(e) {
        if (e.key === 'Enter' && !searchQuery.trim() && suggestedItems && suggestedItems.length > 0) {
            e.preventDefault();
            handleSuggestionClick(suggestedItems[0]);
        }
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
</script>

<svelte:window 
    on:blur={() => handleBlur()} 
    on:resize={() => updateTruncationStatus()}
/>

<div class="app-container">
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
                    placeholder={suggestedItems && suggestedItems.length > 0 ? `回车快速填入: ${suggestedItems[0].name}` : "Search keys..."} 
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
                            class:top-pick={idx === 0}
                            class:hover-expand={hoveredExpandedIdx === idx}
                            on:mouseenter={(e) => handleChipMouseEnter(e, idx)}
                            on:mouseleave={handleChipMouseLeave}
                            on:focus={(e) => handleChipMouseEnter(e, idx)}
                            on:blur={handleChipMouseLeave}
                            on:click={() => handleSuggestionClick(item)}
                            title={item.value || ''}
                        >
                            <span class="chip-title">{item.name}</span>
                            {#if idx === 0}
                                <span class="chip-badge">Enter</span>
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
</div>

{#if globalContextMenu.visible}
  <div 
    class="context-menu"
    style="position: fixed; top: {globalContextMenu.y}px; left: {globalContextMenu.x}px; transform-origin: {globalContextMenu.flipX ? 'right' : 'left'} {globalContextMenu.flipY ? 'bottom' : 'top'};"
    in:scale={{ duration: 130, easing: iosElastic }} out:fade={{ duration: 60 }}
    on:contextmenu|preventDefault>
    {#if globalContextMenu.targetNode?.type === 'folder'}
        <div class="menu-item" on:click={addText} on:keydown={(e => {e.key === 'Enter' && addText()})}>New Text</div>
        <div class="menu-item" on:click={addDir} on:keydown={(e => {e.key === 'Enter' && addDir()})}>New Folder</div>
        <div class="menu-divider"></div>
        <div class="menu-item" on:click={editDir} on:keydown={(e => {e.key === 'Enter' && editDir()})}>Edit</div>
        <div class="menu-divider"></div>
    {/if}
    {#if globalContextMenu.targetNode?.type === 'text'}
        <div class="menu-item" on:click={editText} on:keydown={(e => {})}>Edit</div>
        <div class="menu-divider"></div>
    {/if}
    <div class="menu-item delete" on:click={deleteItem} on:keydown={(e => {})}>Delete</div>
  </div>
{/if}

{#if showDirInput}
    <div class="modal-overlay" on:keyup={cancelAddDir} on:click={cancelAddDir} in:fade={{ duration: 130, easing: quartOut }} out:fade={{ duration: 80 }}>
        <div class="modal-box compact" on:keyup|stopPropagation in:fly={{ y: 15, duration: 230, easing: cubicOut }} out:fly={{ y: 10, duration: 100 }}>
            <input type="text" bind:value={dirName} bind:this={dirInputRef} placeholder="Folder Name" 
                on:click={(e) => e.stopPropagation()}
                on:keydown={(e) => { if (e.key === 'Enter') confirmAddDir(); if (e.key === 'Escape') cancelAddDir(); }}/>
        </div>
    </div>
{/if}

{#if showTextInput}
    <div class="modal-overlay" on:keydown={cancelAddText} on:click={cancelAddText} in:fade={{ duration: 130, easing: quartOut }} out:fade={{ duration: 80 }}>
        <div class="modal-box" on:keydown|stopPropagation on:click|stopPropagation in:fly={{ y: 15, duration: 230, easing: cubicOut }} out:fly={{ y: 10, duration: 100 }}>
            <div class="input-group">
                <input type="text" class="title-input" bind:value={titleName} bind:this={titleInputRef} placeholder="Key / Name" on:keydown={(e) => handleKeyDown(e, true)}/>
                <textarea class="value-input" bind:value={textName} bind:this={textInputRef} placeholder="Value / Command" spellcheck="false" on:keydown={(e) => handleKeyDown(e, false)}></textarea>
            </div>
            <div class="modal-footer">
                <span class="hint">Shift+Enter for newline / Enter to save</span>
            </div>
        </div>
    </div>
{/if}

{#if showSettings}
    <Setting 
        on:close={
        () => {
            showSettings = false;
            ExitSettingsMode();
        }
    } 
    />
{/if}

{#if showDeleteConfirm}
    <div class="modal-overlay" on:click={cancelDelete} on:keydown={cancelDelete} in:fade={{ duration: 130, easing: quartOut }} out:fade={{ duration: 80 }}>
        <div class="modal-box compact confirm-modal" on:keydown|stopPropagation on:click|stopPropagation in:fly={{ y: 15, duration: 230, easing: cubicOut }} out:fly={{ y: 10, duration: 100 }}>
            <div class="confirm-content">
                <div class="confirm-text">
                    <div class="confirm-title">Confirm Delete</div>
                    <div class="confirm-message">
                        Are you sure you want to delete "{itemToDelete?.name}"?
                    </div>
                </div>
            </div>
            <div class="modal-footer confirm-footer">
                <button class="btn btn-cancel" on:click={cancelDelete}>Cancel</button>
                <button class="btn btn-delete" on:click={confirmDeleteItem}>Delete</button>
            </div>
        </div>
    </div>
{/if}

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
        gap: 6px;
        margin-top: 5px;
        padding-top: 4px;
        border-top: 1px dashed rgba(74, 96, 116, 0.12);
        overflow: hidden;
    }

    .suggestion-tag {
        flex-shrink: 0;
        font-size: 11px;
        font-weight: 600;
        color: #5c7080;
        letter-spacing: 0.2px;
        user-select: none;
    }

    .suggestion-chips {
        display: flex;
        align-items: center;
        gap: 5px;
        overflow: hidden;
        flex: 1;
        min-width: 0;
        height: 22px;
    }

    .suggestion-chip {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 6px;
        background: rgba(255, 255, 255, 0.65);
        border: 1px solid rgba(75, 98, 119, 0.15);
        border-radius: 4px;
        font-size: 11px;
        color: #2c3e50;
        cursor: pointer;
        height: 22px;
        box-sizing: border-box;
        white-space: nowrap;
        overflow: hidden;
        /* 默认等分占比 1:1:1 */
        flex: 1 1 0;
        min-width: 28px;
        /* iOS 弹簧贝塞尔曲线 (与 Auto Paste 保持一致) */
        transition: flex 0.38s cubic-bezier(0.34, 1.4, 0.64, 1),
                    background-color 0.2s ease,
                    border-color 0.2s ease,
                    box-shadow 0.2s ease,
                    color 0.2s ease,
                    padding 0.28s ease,
                    opacity 0.2s ease;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
    }

    .suggestion-chip:hover {
        background: rgba(235, 243, 252, 0.9);
        border-color: rgba(59, 130, 246, 0.4);
        color: #1d4ed8;
    }

    .suggestion-chip.top-pick {
        background: rgba(239, 246, 255, 0.85);
        border-color: rgba(59, 130, 246, 0.35);
        color: #1e40af;
        font-weight: 500;
    }

    .suggestion-chip.top-pick:hover {
        background: rgba(219, 234, 254, 0.95);
    }

    /* 当存在展开项时，压缩未悬停的兄弟项 */
    .suggestion-chips.has-expanded .suggestion-chip:not(.hover-expand) {
        flex: 0.5 1 0;
        padding: 2px 4px;
        opacity: 0.82;
    }

    /* 截断项悬停展开：弹性拉伸，占据行内主要空间 */
    .suggestion-chip.hover-expand {
        flex: 3.5 1 0;
        background: rgba(235, 243, 252, 0.95);
        border-color: rgba(59, 130, 246, 0.55);
        color: #1d4ed8;
        box-shadow: 0 2px 8px rgba(59, 130, 246, 0.16);
    }

    .suggestion-chip.hover-expand.top-pick {
        background: rgba(225, 239, 255, 0.98);
        border-color: rgba(59, 130, 246, 0.65);
        color: #1e40af;
    }

    .chip-title {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        flex: 1 1 auto;
        min-width: 0;
        text-align: left;
    }

    .chip-badge {
        flex-shrink: 0;
        font-size: 9px;
        padding: 0 3px;
        background: rgba(59, 130, 246, 0.15);
        color: #2563eb;
        border-radius: 2px;
        font-weight: 600;
        line-height: 12px;
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
        font-size: 13px;
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

    .context-menu {
        background: rgba(248, 250, 252, 0.84);
        -webkit-backdrop-filter: blur(22px) saturate(1.3);
        backdrop-filter: blur(22px) saturate(1.3);
        border: 1px solid rgba(255, 255, 255, 0.74);
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.68),
            0 10px 26px rgba(25, 41, 55, 0.18);
        border-radius: 7px;
        padding: 4px;
        min-width: 128px;
        z-index: 9999;
    }

    .menu-item {
        padding: 4px 10px;
        font-size: 13px;
        border-radius: 4px;
        cursor: pointer;
        color: #333;
        text-align: left;
        transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .menu-item:hover {
        background: rgba(213, 235, 247, 0.78);
        color: #215f82;
        box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.56);
    }

    .menu-item.delete:hover {
        background: #ef4444;
        color: #fff;
    }
    
    .menu-divider {
        height: 1px;
        background: rgba(0,0,0,0.1);
        margin: 4px 0;
    }

    .modal-overlay {
        position: fixed;
        top: 0;left: 0;right: 0;bottom: 0;
        background: rgba(215, 225, 233, 0.28);
        -webkit-backdrop-filter: blur(8px) saturate(1.12);
        backdrop-filter: blur(8px) saturate(1.12);
        display: flex;
        align-items: flex-start;
        justify-content: center;
        padding-top: 80px;
        z-index: 9998;
    }

    .modal-box {
        background: rgba(249, 251, 252, 0.86);
        -webkit-backdrop-filter: blur(26px) saturate(1.25);
        backdrop-filter: blur(26px) saturate(1.25);
        width: 380px;
        border-radius: 8px;
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.72),
            0 14px 38px rgba(24, 39, 52, 0.2);
        border: 1px solid rgba(255, 255, 255, 0.72);
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .modal-box.compact {
        width: 300px;
        padding: 8px;
    }

    .input-group {
        display: flex;
        flex-direction: column;
    }

    .input-group input,
    .input-group textarea {
        box-sizing: border-box;
        border: none;
        padding: 12px 16px;
        font-size: 14px;
        outline: none;
        width: 100%;
        background: transparent;
        color: #293b4a;
        transition: background-color 0.18s ease;
    }

    .input-group textarea {
        min-height: 112px;
        max-height: 220px;
        resize: vertical;
        line-height: 1.45;
        font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
        overflow-y: auto;
        overflow-x: hidden;
    }

    .title-input {
        border-bottom: 1px solid #eee !important;
        font-weight: 500;
    }

    .modal-box.compact input {
        border: 1px solid #eee;
        border-radius: 4px;
        padding: 6px 10px;
        background: #f9f9f9;
    }
    .modal-box.compact input:focus {
        background: #fff;
        border-color: #3b82f6;
    }

    .modal-footer {
        background: #f9fafb;
        padding: 6px 10px;
        text-align: right;
        border-top: 1px solid #f0f0f0;
    }

    .confirm-modal {
        min-width: 25px;
        padding: 12px;
    }

    .confirm-content {
        display: flex;
        align-items: flex-start;
        gap: 15px;
        margin-bottom: 20px;
    }

    .confirm-text {
        flex: 1;
    }

    .confirm-title {
        font-size: 16px;
        font-weight: 600;
        color: #333;
        margin-bottom: 8px;
    }

    .confirm-message {
        font-size: 13px;
        color: #666;
        line-height: 1.5;
        word-break: break-all;
    }

    .confirm-footer {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        padding: 0;
        background: transparent;
        border-top: none;
    }

    .btn {
        padding: 8px 16px;
        border-radius: 4px;
        font-size: 13px;
        cursor: pointer;
        border: 1px solid transparent;
        transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .btn-cancel {
        background: #f5f5f5;
        color: #666;
        border-color: #ddd;
    }

    .btn-cancel:hover {
        background: #e5e5e5;
        color: #333;
    }

    .btn-delete {
        background: #fee2e2;
        color: #dc2626;
        border-color: #fecaca;
    }

    .btn-delete:hover {
        background: #fecaca;
        color: #b91c1c;
    }

    .hint {
        font-size: 11px;
        color: #999;
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
