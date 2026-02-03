<script>
    import { onMount, tick, onDestroy } from 'svelte';
    import { fade, fly} from 'svelte/transition';
    import { GetContent, SaveContent } from '../wailsjs/go/main/App'; 
    import { LogInfo, Quit    } from '../wailsjs/runtime';
    import TreeItem from './components/TreeItem.svelte';

    let data = [];
    let expanded = {};
    let showMenu = false;
    let isHovered = false;

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

		try {
			// 获取父级路径
			const keys = globalContextMenu.targetKey.split(".");
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

			// 隐藏菜单
			hideContextMenu();
            cleanGlobalContextMenu();
		} catch (err) {
			console.error("删除失败", err);
		}
	}

    onMount(() => {
        document.addEventListener('click', handleGlobalClick);
        document.addEventListener('contextmenu', hideContextMenu); // 右键其他地方也关闭
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

    // 表单验证
    $: isFormValid = titleName.trim() !== "" && textName.trim() !== "";

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
        if (!isFormValid) {
            if (!titleName.trim()) {
                titleInputRef?.focus();
                alert("请输入名称");
            } else if (!textName) {
                textInputRef?.focus();
                alert("请输入值");
            }
            return;
        }

        if (titleName.trim() !== "" && textName.trim() !== "") {
            if (globalContextMenu.isFolder) {
                // 文件夹下添加文本
                const folderKey = globalContextMenu.targetKey; // 例如 "0.士大夫大师傅.0.士大夫"
                const pathParts = folderKey.split('.');
                
                // 递归访问嵌套结构
                let current = data;
                for (let i = 0; i < pathParts.length; i += 2) {
                    const index = parseInt(pathParts[i]);
                    const key = pathParts[i + 1];
                    
                    // 获取当前层级的数组
                    const arr = current[index];
                    
                    // 获取数组中对应 key 的值（应该是一个数组）
                    current = arr[key];
                }
                
                // 现在 current 指向目标文件夹的数组
                if (Array.isArray(current)) {
                    const newDir = { [titleName.trim()]: textName };
                    current.push(newDir);
                }
        } else {
            // 根目录添加文本
            const newDir = { [titleName.trim()]: textName };
            data = [...data, newDir];
        }

        
        updateData(data);
        cleanGlobalContextMenu();
        titleName = "";
        textName = "";
        showTextInput = false;
    }
    }

    function cancelAddText() {
        titleName = "";
        textName = "";
        showTextInput = false;
        cleanGlobalContextMenu();
    }

    // 自动聚焦
    $: if (showTextInput && titleInputRef) {
        setTimeout(() => titleInputRef.focus(), 0);
    }

    function addDir() {
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
        const dirNameTrim = dirName.trim();
        if (dirName.trim() !== "") {
            if (globalContextMenu.isFolder) {
                const folderKey = globalContextMenu.targetKey; // 例如 "0.士大夫大师傅.0.士大夫"
                const pathParts = folderKey.split('.'); // ["0", "士大夫大师傅", "0", "士大夫"]

                // 递归访问嵌套结构
                let current = data;
                for (let i = 0; i < pathParts.length; i += 2) {
                    const index = parseInt(pathParts[i]);
                    const key = pathParts[i + 1];
                    
                    // 获取当前层级的数组
                    const arr = current[index];
                    
                    // 获取数组中对应 key 的值（应该是一个数组）
                    current = arr[key];
                }
                
                // 现在 current 指向目标文件夹的数组
                if (Array.isArray(current)) {
                    const newDir = { [dirNameTrim]: [] };
                    current.push(newDir);
                }
            } else {
                // 在根目录添加文件夹
                const newDir = { [dirNameTrim]: [] };
                data = [...data, newDir];
            }
            
            updateData(data);
        }
        showDirInput = false;
    }

    function cancelAddDir() {
        showDirInput = false;
        dirName = "";
        cleanGlobalContextMenu();
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
        const now = Date.now();
        if (now - lastFocusTime < 200) {
            return;
        }else{
            Quit();
        }
    }
    // ----------------------------------------------

    function updateData(newData) {
        data = newData;
        SaveContent(data);
    }

</script>


<svelte:window 
    on:blur={() => handleBlur()} 
    on:focus={() => handleFocus()}
/>

<!-- 整体容器 -->
<div class="app-container">
    
    <!-- 极简头部：类似于 VS Code 或 Mac Spotlight -->
    <div class="sticky-header">
        <div class="header-row">
            <span class="app-title" on:click={() => Quit()} on:keyup={(e) => {
                if (e.key === 'Enter') {
                    Quit();
                }
            }}>Quick
            Clip</span>
            
            <div class="search-wrapper">
                <input type="search" class="search-input" placeholder="Search...">
            </div>

            <div class="action-wrapper">
                <button class="icon-btn add-btn" on:click={toggleMenu} title="New Item">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="12" y1="5" x2="12" y2="19"></line>
                        <line x1="5" y1="12" x2="19" y2="12"></line>
                    </svg>
                </button>
                {#if showMenu}
                    <div class="dropdown-menu" on:click|stopPropagation on:keydown|stopPropagation transition:fade={{duration: 100}}>
                        <button on:click={addText}>文本 (Text)</button>
                        <button on:click={addDir}>文件夹 (Folder)</button>
                    </div>
                {/if}
            </div>
        </div>
    </div>
    
    <!-- 内容区域 -->
    <div class="content-scrollable">
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
    </div>
</div>

<!-- 上下文菜单 -->
{#if globalContextMenu.visible}
  <div 
    class="context-menu"
    style="position: fixed; top: {globalContextMenu.y}px; left: {globalContextMenu.x}px;"
    on:contextmenu|preventDefault>
    {#if globalContextMenu.isFolder}
        <div class="menu-item" on:click={addText} on:keyup={(e) => {
                if (e.key === 'Enter') {
                    Quit();
                }
            }}>New Text</div>
        <div class="menu-item" on:click={addDir} on:keyup={(e) => {
                if (e.key === 'Enter') {
                    Quit();
                }
            }}>New Folder</div>
        <div class="menu-divider"></div>
    {/if}
    <div class="menu-item delete" on:click={deleteItem} on:keyup={(e) => {
                if (e.key === 'Enter') {
                    Quit();
                }
            }}>Delete</div>
  </div>
{/if}

<!-- 目录弹窗 -->
{#if showDirInput}
    <div class="modal-overlay" on:keyup={cancelAddDir} on:click={cancelAddDir} transition:fade={{duration: 80}}>
        <div class="modal-box compact" on:keyup|stopPropagation transition:fly={{ y: -10, duration: 150 }}>
            <input type="text" bind:value={dirName} bind:this={dirInputRef} placeholder="Folder Name" 
                on:keydown={(e) => { if (e.key === 'Enter') confirmAddDir(); if (e.key === 'Escape') cancelAddDir(); }}/>
        </div>
    </div>
{/if}

<!-- 文本弹窗 -->
{#if showTextInput}
    <div class="modal-overlay" on:keydown={cancelAddText} on:click={cancelAddText} transition:fade={{duration: 80}}>
        <div class="modal-box" on:keydown|stopPropagation on:click|stopPropagation transition:fly={{ y: -10, duration: 150 }}>
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

<style>
    /* 主容器 */
    .app-container {
        width: 100vw; /* 占满窗口 */
        height: 100vh;
        background: rgba(255, 255, 255, 0.92); /* 略微不透明一点，提高阅读性 */
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    /* --- 头部样式：极致紧凑 --- */
    .sticky-header {
        flex-shrink: 0;
        background: rgba(245, 245, 245, 0.85); /* 浅灰色背景区分头部 */
        backdrop-filter: blur(20px);
        border-bottom: 1px solid rgba(0,0,0,0.08);
        padding: 8px 10px 3px;
        z-index: 10;
    }

    .header-row {
        display: flex;
        align-items: center;
        gap: 8px;
        height: 28px; /* 强制高度 */
    }

    .app-title {
        font-weight: 600;
        font-size: 13px;
        color: #444;
        white-space: nowrap;
        cursor: default;
        user-select: none;
        margin-right: 4px;
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
        transition: all 0.2s;
    }

    .search-input:focus {
        border-color: #3b82f6; /* 聚焦蓝 */
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
        transition: background 0.1s;
        padding: 0;
    }

    .icon-btn:hover {
        background: rgba(0,0,0,0.06);
        color: #000;
    }

    .icon-btn svg {
        opacity: 0.8;
    }

    /* 下拉菜单 */
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
    }

    .dropdown-menu button:hover {
        background: #3b82f6;
        color: white;
    }

    /* --- 列表区域 --- */
    .content-scrollable {
        flex: 1;
        overflow-y: auto;
        padding: 4px 0; /* 极小内边距 */
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

    /* 
       重要：这里使用了 :global 来强制覆盖 TreeItem 内部可能的样式 
       目标是让列表看起来像 VS Code 的侧边栏，而不是卡片
    */
    :global(.tree-root ul) {
        list-style: none;
        padding-left: 16px; /* 缩进 */
        margin: 0;
        border-left: 1px solid rgba(0,0,0,0.05); /* 淡淡的引导线 */
    }

    :global(.tree-root li) {
        margin: 0;
        padding: 0;
    }

    /* 模拟 TreeItem 内容行的样式 (你需要确保 TreeItem 内部结构能匹配或调整) */
    :global(.tree-item-content) {
        display: flex;
        align-items: center;
        padding: 4px 8px; /* 紧凑行高 */
        cursor: pointer;
        border-radius: 4px;
        margin: 1px 4px;
        color: #333;
    }

    :global(.tree-item-content:hover) {
        background-color: rgba(0,0,0,0.04);
    }
    
    /* 选中的样式 (如果有) */
    :global(.tree-item-content.selected) {
        background-color: #e0e7ff;
        color: #3730a3;
    }

    /* --- 上下文菜单 (Native Look) --- */
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
    }

    .menu-item:hover {
        background: #3b82f6;
        color: #fff;
    }

    .menu-item.delete:hover {
        background: #ef4444; /* 红色警告 */
    }
    
    .menu-divider {
        height: 1px;
        background: rgba(0,0,0,0.1);
        margin: 4px 0;
    }

    /* --- 弹窗 (Spotlight 风格) --- */
    .modal-overlay {
        position: fixed;
        top: 0;left: 0;right: 0;bottom: 0;
        background: rgba(255,255,255,0.5); /* 非常淡的遮罩 */
        z-index: 2000;
        display: flex;
        align-items: flex-start; /* 靠上显示 */
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

    .hint {
        font-size: 11px;
        color: #999;
    }

</style>