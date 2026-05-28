<script>
    import { onMount, tick, onDestroy } from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { quartOut, cubicOut, backOut } from 'svelte/easing';
    import { EnterSettingsMode, GetContent, SaveContent, ExitSettingsMode, ToggleWindow, HideWindow} from '../wailsjs/go/main/App'; 
    import { LogInfo, Quit, EventsOn   } from '../wailsjs/runtime';
    import TreeItem from './components/TreeItem.svelte';
    import Setting from './components/Setting.svelte';
    // 强弹性 (0.34, 1.56, 0.64, 1) — 主要点击交互（按钮、菜单项、开关）
    // 中弹性 (0.34, 1.3, 0.64, 1) — 功能性元素（搜索框、结果列表）
    // 弱弹性 (0.34, 1.15, 0.64, 1) — 辅助性过渡（边框、阴影变化）
    let data = [];
    let expanded = {};
    let showMenu = false;
    let isHovered = false;
    let showSettings = false;

    // 粘贴模式开关: true=Auto Paste, false=Not Paste
    let autoPaste = true;

    // 编辑模式
    let isEditMode = false; // 标记当前是编辑还是新增
    let editingPath = "";

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

    // 在父组件中（例如 App.svelte）
    let globalContextMenu = {
        visible: false,
        x: 0,
        y: 0,
        targetKey: null,
        targetValue: null,
        isFolder: false
    };

    function cleanGlobalContextMenu() {
        globalContextMenu = {
            visible: false,
            x: 0,
            y: 0,
            targetKey: null,
            targetValue: null,
            isFolder: false
        };
    }

    function showContextMenu(e, key, val, isFolder) {
        e.preventDefault();
        e.stopPropagation();
        
        globalContextMenu = {
            visible: true,
            x: e.pageX,
            y: e.pageY,
            targetKey: key,
            targetValue: val,
            isFolder: isFolder
        };
    }

    function hideContextMenu() {
        globalContextMenu.visible = false;
    }

    // 点击页面其他地方关闭菜单
    function handleGlobalClick(e) {
        if (!e.target.closest('.context-menu')) {
            hideContextMenu();
        }

        if (!e.target.closest('.dropdown-menu') && !e.target.closest('.add-btn') && !e.target.closest('.icon-btn')) {
            cancelMenu();
        }
    }

    function deleteItem() {
		if (!globalContextMenu.targetKey) return;

		// 保存要删除的项目信息
		itemToDelete = {
			key: globalContextMenu.targetKey,
			isFolder: globalContextMenu.isFolder
		};

		// 显示确认弹窗
		showDeleteConfirm = true;
		// 隐藏右键菜单
		hideContextMenu();
	}

	function confirmDeleteItem() {
		if (!itemToDelete) return;

		try {
			// 获取父级路径
			const keys = itemToDelete.key.split(".");
			const propertyToDelete = keys.pop(); // 要删除的属性名
			const parentPath = keys.join(".");

			let parentObj = data;
			if (parentPath !== "") {
				const parentKeys = parentPath.split(".");
				for (let k of parentKeys) {
					if (parentObj && parentObj[k] !== undefined) {
						parentObj = parentObj[k];
					}
				}
			}

			// 删除属性
			if (parentObj && parentObj.hasOwnProperty(propertyToDelete)) {
				delete parentObj[propertyToDelete];

				// 更新数据
				if (parentPath === "") {
					updateData({ ...data });
				} else {
					updateData([...data]); // 如果是数组，需要新数组引用
				}
			}

			// 隐藏弹窗
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
    const settingsEventListener = async (data) => {
        showSettings = true;
        EnterSettingsMode();
        ToggleWindow();
    };

    // 监听来自后端的 update-content 事件
    const contentEventListener = async (payload) => {
        try {
            LogInfo("update-content发送成功")
            const newData = await GetContent();
            data = newData;
            await tick();
        } catch (error) {
            console.error('Failed to load content:', error);
        }
    };

    onMount(() => {
        document.addEventListener('click', handleGlobalClick);
        document.addEventListener('contextmenu', hideContextMenu); // 右键其他地方也关闭

        EventsOn("show-settings", settingsEventListener);
        EventsOn("update-content", contentEventListener);
        
    });

    onMount(async () => {
        try {
            data = await GetContent();
        } catch (error) {
            console.error('Failed to load content:', error);
        }
    });

    onDestroy(() => {
        document.removeEventListener('click', handleGlobalClick);
        document.removeEventListener('contextmenu', hideContextMenu);

    });

    function toggleExpand(key) {
        expanded[key] = !expanded[key];
    }

    function toggleMenu() {
        showMenu = !showMenu;
    }

    function addText() {
        isEditMode = false; // 新增模式
        editingPath = ""; // 清空路径
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

    // 表单验证: titleName不包含. textName不包含.
    $: isFormValid = titleName.trim() !== "" && textName.trim() !== "" && !titleName.includes(".");

    // 键盘事件处理函数
    function handleKeyDown(event, isTitleInput) {
        const { key } = event;

        if (key === 'Enter') {
            event.preventDefault();

            if (isTitleInput) {
                textInputRef?.focus();
            } else {
                if (isFormValid) {
                    confirmAddText();
                }
            }
        } else if (key === 'Escape') {
            cancelAddText();
        } else if (key === 'Tab' && isTitleInput) {
            if (event.shiftKey === false) {
                event.preventDefault();
                textInputRef?.focus();
            }
        }
    }

    function confirmAddText() {
        // 简单校验
        if (!titleName.trim() || !textName) {
            alert("请完善输入");
            return;
        }
        if (titleName.includes(".")) {
            alert("名称不能包含.");
            return;
        }

        const newKey = titleName.trim();
        const newVal = textName;

        if (isEditMode) {
            // --- 编辑逻辑 ---
            try {
                // 1. 获取父数组、索引、旧名称
                const { parentArr, targetIndex, oldKey } = getParentArrayAndIndex(editingPath);
                
                // 2. 找到该对象
                const itemObj = parentArr[targetIndex];
                
                // 3. 修改逻辑：如果是改名，需要 delete 旧 key
                if (oldKey !== newKey) {
                    delete itemObj[oldKey];
                }
                // 4. 写入新 Key-Value
                itemObj[newKey] = newVal;

                // 强制更新视图
                if (editingPath.startsWith("0.") || !editingPath.includes(".")) {
                   // 根目录稍微特殊，直接全量更新最稳
                   data = [...data]; 
                }
            } catch (e) {
                console.error("编辑失败", e);
            }
        } else {
            // --- 新增逻辑 (保持原样) ---
            const newItem = { [newKey]: newVal };
            if (globalContextMenu.isFolder) {
                // 如果是在文件夹上右键新增
                // 需要解析 globalContextMenu.targetKey 找到那个文件夹数组
                const pathParts = globalContextMenu.targetKey.split('.');
                let current = data;
                for (let i = 0; i < pathParts.length; i += 2) {
                    current = current[parseInt(pathParts[i])][pathParts[i+1]];
                }
                current.push(newItem);
            } else {
                // 根目录新增
                data = [...data, newItem];
            }
        }

        updateData(data); // 保存
        cancelAddText();
    }

    function cancelAddText() {
        titleName = "";
        textName = "";
        showTextInput = false;
        cleanGlobalContextMenu();
    }

    function editText() {
        isEditMode = true;
        editingPath = globalContextMenu.targetKey; // 保存完整路径：0.FolderA.1.KeyName
        showTextInput = true;
        
        // 【修复点1】只截取最后一段作为名称显示
        titleName = globalContextMenu.targetKey.split('.').pop(); 
        textName = globalContextMenu.targetValue;
        
        showMenu = false;
        hideContextMenu();

        tick().then(() => titleInputRef?.focus());
    }

    // 自动聚焦
    $: if (showTextInput && titleInputRef) {
        setTimeout(() => titleInputRef.focus(), 0);
    }

    function addDir() {
        isEditMode = false; // 新增模式
        editingPath = ""; // 清空路径
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
        if (!newDirName || newDirName.includes(".")) {
            alert("名称无效");
            return;
        }

        if (isEditMode) {
            // --- 编辑逻辑 ---
            try {
                const { parentArr, targetIndex, oldKey } = getParentArrayAndIndex(editingPath);
                const itemObj = parentArr[targetIndex];

                // 文件夹只改名字，必须保留原来的子内容(Value)
                const children = itemObj[oldKey]; 

                if (oldKey !== newDirName) {
                    delete itemObj[oldKey];      // 删除旧名
                    itemObj[newDirName] = children; // 赋给新名，内容不变
                }
            } catch (e) {
                console.error("文件夹编辑失败", e);
            }
        } else {
            // --- 新增逻辑 (保持原样) ---
            const newItem = { [newDirName]: [] };
            if (globalContextMenu.isFolder) {
                 // 逻辑同 Text 新增
                const pathParts = globalContextMenu.targetKey.split('.');
                let current = data;
                for (let i = 0; i < pathParts.length; i += 2) {
                    current = current[parseInt(pathParts[i])][pathParts[i+1]];
                }
                current.push(newItem);
            } else {
                data = [...data, newItem];
            }
        }

        updateData(data);
        showDirInput = false;
    }

    function cancelAddDir() {
        showDirInput = false;
        dirName = "";
        cleanGlobalContextMenu();
    }

    function editDir() {
        isEditMode = true;
        editingPath = globalContextMenu.targetKey; // 保存完整路径
        showDirInput = true;
        
        // 【修复点1】只显示名称，不显示路径
        dirName = globalContextMenu.targetKey.split('.').pop();
        
        showMenu = false;
        hideContextMenu();

        tick().then(() => dirInputRef?.focus());
    }

    function cancelMenu() {
        showMenu = false;
    }

    // 焦点--------------------------------------------
    let lastFocusTime = 0;

    function handleFocus() {
        lastFocusTime = Date.now();
    }

    function handleBlur() {
        requestAnimationFrame(() => {
            if (document.hasFocus()) {
                return;
            }

            if (showTextInput || showDirInput || showSettings) {
                return; 
            }

            HideWindow();
        });
}
    // ----------------------------------------------

    function updateData(newData) {
        data = newData;
        SaveContent(data);
    }

    function getParentArrayAndIndex(pathStr) {
        const parts = pathStr.split('.');
        const keyName = parts.pop();
        const indexStr = parts.pop();
        const index = parseInt(indexStr);

        let currentArr = data;
        
        if (parts.length > 0) {
            for (let i = 0; i < parts.length; i += 2) {
                const pIndex = parseInt(parts[i]);
                const pKey = parts[i+1];
                currentArr = currentArr[pIndex][pKey];
            }
        }
        
        return { parentArr: currentArr, targetIndex: index, oldKey: keyName };
    }

    import { PasteAndHide } from '../wailsjs/go/main/App';
    let searchQuery = "";
    let searchResults = [];

    function performSearch(items, query, path = "") {
        if (!query.trim()) return [];
        let results = [];
        const q = query.toLowerCase();

        for (const item of items) {
            for (const [key, val] of Object.entries(item)) {
                if (Array.isArray(val)) {
                    results = [...results, ...performSearch(val, query, path + key + " > ")];
                } else {
                    if (key.toLowerCase().includes(q)) {
                        results.push({
                            name: key,
                            content: val,
                            fullPath: path + key
                        });
                    }
                }
            }
        }
        return results;
    }

    $: {
        if (searchQuery.trim()) {
            searchResults = performSearch(data, searchQuery);
        } else {
            searchResults = [];
        }
    }

    function handleSearchResultClick(content) {
        navigator.clipboard.writeText(content).then(() => {
            PasteAndHide();
            searchQuery = "";
        }).catch(err => console.error("Search copy failed:", err));
    }

</script>

<svelte:window 
    on:blur={() => handleBlur()} 
    on:focus={() => handleFocus()}
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
                    placeholder="Search keys..." 
                    bind:value={searchQuery}
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
                    <div class="dropdown-menu" on:click|stopPropagation on:keydown|stopPropagation in:fly={{ y: -4, duration: 130, easing: backOut }} out:fade={{duration: 80}}>
                        <button on:click={addText}>文本 (Text)</button>
                        <button on:click={addDir}>文件夹 (Folder)</button>
                    </div>
                {/if}
            </div>
        </div>
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
                    {#each data as item, index (index)}
                        <TreeItem 
                            itemKey={index.toString()} 
                            value={item} 
                            {data} 
                            {updateData} 
                            {expanded} 
                            {toggleExpand} 
                            index={index} 
                            showContextMenu={showContextMenu}
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
    style="position: fixed; top: {globalContextMenu.y}px; left: {globalContextMenu.x}px;"
    on:contextmenu|preventDefault>
    {#if globalContextMenu.isFolder}
        <div class="menu-item" on:click={addText} on:keydown={(e => {e.key === 'Enter' && addText()})}>New Text</div>
        <div class="menu-item" on:click={addDir} on:keydown={(e => {e.key === 'Enter' && addDir()})}>New Folder</div>
        <div class="menu-divider"></div>
        <div class="menu-item" on:click={editDir} on:keydown={(e => {e.key === 'Enter' && editDir()})}>Edit</div>
        <div class="menu-divider"></div>
    {/if}
    {#if !globalContextMenu.isFolder}
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
                <input type="text" class="value-input" bind:value={textName} bind:this={textInputRef} placeholder="Value / Content" on:keydown={(e) => handleKeyDown(e, false)}/>
            </div>
            <div class="modal-footer">
                <span class="hint">Tab to change box / Enter to save</span>
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
                        Are you sure?
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
        width: 100vw;
        height: 100vh;
        background: rgba(255, 255, 255, 0.92);
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .sticky-header {
        flex-shrink: 0;
        background: rgba(245, 245, 245, 0.85);
        backdrop-filter: blur(20px);
        border-bottom: 1px solid rgba(0,0,0,0.08);
        padding: 8px 10px 3px;
        z-index: 10;
    }

    .header-row {
        display: flex;
        align-items: center;
        gap: 8px;
        height: 28px;
    }

    .search-wrapper {
        flex: 1;
        position: relative;
    }

    .search-input {
            width: 100%;
            height: 26px;
            border: 1px solid rgba(0,0,0,0.1);
            background: #fff;
            border-radius: 4px;
            padding: 0 8px;
            font-size: 13px;
            outline: none;
            transition: all 0.3s cubic-bezier(0.34, 1.3, 0.64, 1);
        }

    .search-input:focus {
        border-color: #3b82f6;
        box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
    }

    .action-wrapper {
        position: relative;
    }

    .icon-btn {
        background: transparent;
        border: 1px solid transparent;
        border-radius: 4px;
        width: 26px;
        height: 26px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #555;
        cursor: pointer;
        transition: all 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
        padding: 0;
    }

    .icon-btn:hover {
        background: rgba(0,0,0,0.06);
        color: #000;
        transform: scale(1.08);
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
        border: 1px solid rgba(0,0,0,0.06);
        border-radius: 20px;
        background: rgba(255,255,255,0.5);
        cursor: pointer;
        user-select: none;
        outline: none;
        font-family: inherit;
        font-size: 13px;
        white-space: nowrap;
        margin-right: 4px;
        flex-shrink: 0;
        transition: all 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
        overflow: hidden;
    }

    .paste-toggle:hover {
        gap: 8px;
        padding: 2px 8px 2px 10px;
        background: rgba(255,255,255,0.85);
        border-color: rgba(0,0,0,0.12);
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
        background: #fff;
        border: 1px solid rgba(0,0,0,0.1);
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        border-radius: 6px;
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
        background: #3b82f6;
        color: white;
    }

    .content-scrollable {
        flex: 1;
        overflow-y: auto;
        padding: 4px 0;
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

    :global(.tree-root ul) {
        list-style: none;
        padding-left: 16px;
        margin: 0;
        border-left: 1px solid rgba(0,0,0,0.05);
    }

    :global(.tree-root li) {
        margin: 0;
        padding: 0;
    }

    :global(.tree-item-content) {
        display: flex;
        align-items: center;
        padding: 4px 8px;
        cursor: pointer;
        border-radius: 4px;
        margin: 1px 4px;
        color: #333;
    }

    :global(.tree-item-content:hover) {
        background-color: rgba(0,0,0,0.04);
    }
    
    :global(.tree-item-content.selected) {
        background-color: #e0e7ff;
        color: #3730a3;
    }

    .context-menu {
        background: rgba(255, 255, 255, 0.95);
        backdrop-filter: blur(10px);
        border: 1px solid rgba(0,0,0,0.1);
        box-shadow: 0 6px 16px rgba(0,0,0,0.12);
        border-radius: 6px;
        padding: 4px;
        min-width: 140px;
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
        background: #3b82f6;
        color: #fff;
    }

    .menu-item.delete:hover {
        background: #ef4444;
    }
    
    .menu-divider {
        height: 1px;
        background: rgba(0,0,0,0.1);
        margin: 4px 0;
    }

    .modal-overlay {
        position: fixed;
        top: 0;left: 0;right: 0;bottom: 0;
        background: rgba(255,255,255,0.5);
        z-index: 2000;
        display: flex;
        align-items: flex-start;
        justify-content: center;
        padding-top: 80px;
        z-index: 9998;
    }

    .modal-box {
        background: #fff;
        width: 380px;
        border-radius: 8px;
        box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        border: 1px solid rgba(0,0,0,0.1);
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

        .input-group input {
        border: none;
        padding: 12px 16px;
        font-size: 14px;
        outline: none;
        width: 100%;
        background: transparent;
        transition: background 0.3s cubic-bezier(0.34, 1.3, 0.64, 1);
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
        padding: 5px;
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
        background: #fff;
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
        background: rgba(59, 130, 246, 0.1);
        padding-left: 20px;
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
