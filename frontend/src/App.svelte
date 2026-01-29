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

        if (globalContextMenu.isFolder) {
            // 文件夹下添加文本
            const folderKey = globalContextMenu.targetKey.split(".")[1];
            // 找到对应的文件夹对象并获取其键名
            const folderObj = data.find(item => typeof item === 'object' && item[folderKey]);
            if (folderObj) {
                // 获取文件夹内的数组
                const folderArray = folderObj[folderKey];
                if (Array.isArray(folderArray)) {
                    // 向文件夹内的数组添加新的文本项
                    folderArray.push({ [titleName.trim()]: textName });
                }
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
                // 在文件夹内添加新文件夹
                const folderKey = globalContextMenu.targetKey.split(".")[1];
                // 找到对应的文件夹对象
                const folderObj = data.find(item => typeof item === 'object' && item[folderKey]);
                if (folderObj) {
                    // 获取文件夹内的数组
                    const folderArray = folderObj[folderKey];
                    if (Array.isArray(folderArray)) {
                        // 向文件夹内的数组添加新的文件夹项
                        const newDir = { [dirNameTrim]: [] };
                        folderArray.push(newDir);
                    }
                }
            } else {
                // 在根目录添加文件夹
                const newDir = { [dirNameTrim]: [] };
                data = [...data, newDir];
            }
            
            updateData(data);
        }
        showDirInput = false;
        cleanGlobalContextMenu();
    }

    function cancelAddDir() {
        showDirInput = false;
        dirName = "";
        cleanGlobalContextMenu();
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

<h3 class="main-title" 
    on:mouseenter={() => isHovered = true} 
    on:mouseleave={() => isHovered = false}
    on:click={() => Quit()}
    on:keydown={event => {
        if (event.key === 'Escape') {
            Quit();
        }
    }}
    class:has-hover={isHovered}
    style="pointer-events: auto;">
    Quick-Clip
</h3>
<div class="app-container" class:has-hover={isHovered}>
    <!-- 使用 sticky 容器包裹顶部控件 -->
    <div class="sticky-header">
        <div class="top-controls">
            <button class="add-btn" on:click={toggleMenu}>+</button>
            {#if showMenu}
                <div class="add-menu" transition:fade={{duration: 100}}>
                    <button on:click={addText}>添加文本</button>
                    <button on:click={addDir}>添加目录</button>
                </div>
            {/if}
            <input type="search" class="search-input" placeholder="搜索...">
        </div>
    </div>
    
    <div class="content-scrollable">
        {#if data.length === 0}
            <p class="loading">Loading...</p>
        {:else}
            <ul class="content-list">
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

<!-- 在 App.svelte 中渲染全局菜单 -->
{#if globalContextMenu.visible}
  <div 
    class="context-menu"
    style="position: fixed; top: {globalContextMenu.y}px; left: {globalContextMenu.x}px; z-index: 9999;"
    on:contextmenu|preventDefault>
    <ul class="menu-list">
      
      {#if globalContextMenu.isFolder}
        <li class="menu-item" 
        on:click={addText}
        on:keydown={event => {
          if (event.key === 'Enter') {
              addText();
          }
        }}
        >添加文本</li>

        <li class="menu-item" 
        on:click={addDir}
        on:keydown={event => {
          if (event.key === 'Enter') {
              addDir();
          }
        }}
        >添加目录</li>
      {/if}

      <li class="menu-item" 
      on:click={deleteItem}
      on:keydown={event => {
        if (event.key === 'Enter') {
            deleteItem();
        }
      }}
      >删除</li>
      <!-- 可以添加更多菜单项 -->
    </ul>
  </div>
{/if}

<!-- 新增：目录名称输入弹窗 -->
{#if showDirInput}
    <div class="modal-overlay" on:keyup={cancelAddDir} transition:fade={{duration: 50}}>
        <div class="dir-input-modal" 
        on:keyup|stopPropagation 
        transition:fly={{ 
                 y: -15, 
                 duration: 100
             }}>
            <input 
                type="text" 
                bind:value={dirName}
                bind:this={dirInputRef}
                placeholder="目录名称"
                on:keydown={(e) => {
                    if (e.key === 'Enter') confirmAddDir();
                    if (e.key === 'Escape') cancelAddDir();
                }}
            />
            <div class="modal-buttons">
                <button class="cancel-btn" on:click={cancelAddDir}>取消</button>
                <button class="confirm-btn" on:click={confirmAddDir} disabled={!dirName.trim()}>确定</button>
            </div>
        </div>
    </div>
{/if}

{#if showTextInput}
    <!-- 点击遮罩层关闭 -->
    <div class="modal-overlay"
         on:keydown={cancelAddText}
         transition:fade={{duration: 50}}>

        <!-- 内容区域：阻止点击冒泡 -->
        <div class="dir-input-modal"
             on:keydown|stopPropagation
             transition:fly={{ y: -15, duration: 100 }}>

            <!-- 标题输入框 -->
            <input
                type="text"
                bind:value={titleName}
                bind:this={titleInputRef}
                placeholder="名称"
                on:keydown={(e) => handleKeyDown(e, true)}
            />

            <!-- 值输入框 -->
            <input
                type="text"
                bind:value={textName}
                bind:this={textInputRef}
                placeholder="值"
                on:keydown={(e) => handleKeyDown(e, false)}
            />

            <!-- 按钮组 -->
            <div class="modal-buttons">
                <button class="cancel-btn" on:click={cancelAddText}>取消</button>
                <button class="confirm-btn"
                        on:click={confirmAddText}
                        disabled={!isFormValid}>
                    确定
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .app-container {
        background: rgba(255, 255, 255, 0.8);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        border-radius: 6px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        width: 600px;
        max-width: 600px;
        text-align: left;
        transition: all 0.3s ease;
        margin: 0 auto;
        position: relative;
        z-index: 2;
        max-height: 80vh;
        transform: translateY(0);
        /* 修改为 flex 布局 */
        display: flex;
        flex-direction: column;
        overflow: hidden; /* 防止整个容器滚动 */
    }

    .app-container.has-hover {
        transform: translateY(20px);
    }

    .main-title.has-hover {
        transform: translateX(-50%) translateY(5px);
    }

    .loading {
        text-align: center;
        font-size: 1.2rem;
        color: #666;
        padding: 20px;
    }

    /* 新增：sticky 头部容器 */
    .sticky-header {
        position: sticky;
        top: 0;
        z-index: 10;
        background: rgba(255, 255, 255, 0.9);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        padding: 1.5% 2% 0;
        /* 添加底部边框作为分隔线 */
        border-bottom: 1px solid rgba(0, 0, 0, 0.05);
        /* 添加轻微的阴影，增强视觉分离效果 */
        box-shadow: 0 2px 2px rgba(194, 194, 194, 0.204);
    }

    /* 可滚动内容区域 */
    .content-scrollable {
        flex: 1;
        overflow-y: auto;
        padding: 0 2%;
        /* 添加顶部 padding 来补偿 sticky 头部的高度 */
        padding-top: 6px;
    }

    .content-list {
        list-style: none;
        padding: 0;
        margin: 0;
        padding-bottom: 20px;
    }

    .main-title {
        position: absolute;
        top: 5px;
        left: 50%; 
        transform: translateX(-50%) translateY(0);
        width: fit-content; 
        white-space: nowrap; 
        overflow: hidden;    
        text-overflow: ellipsis; 
        font-size: 1.6rem;
        font-weight: 400;
        color: #333;
        margin: 0;
        z-index: 1;
        min-width: 100px;
        pointer-events: auto;
        transition: all 0.3s ease;
    }

    .top-controls {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 5px;
    }

    .add-btn {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.8);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        border: 1px solid rgba(0, 0, 0, 0.1);
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
        color: #333;
        font-size: 24px;
        font-weight: 300;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0; /* 防止按钮被压缩 */
    }

    .add-btn:hover {
        background: rgba(255, 255, 255, 1);
        box-shadow: 0 6px 24px rgba(0, 0, 0, 0.05);
        transform: translateY(-2px);
    }

    .search-input {
        flex: 1;
        margin-left: 10px;
        padding: 10px 0px 10px 8px;
        border-radius: 4px;
        background: rgba(255, 255, 255, 0.5);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        border: 1px solid rgba(0, 0, 0, 0.1);
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
        font-family: inherit;
        font-size: 16px;
        color: #333;
        outline: none;
        transition: all 0.3s ease;
        height: 30px;
        min-width: 0; /* 防止 flex item 溢出 */
    }

    .search-input:focus {
        background: rgba(255, 255, 255, 1);
        box-shadow: 0 6px 24px rgba(0, 0, 0, 0.15);
        border-color: rgba(0, 0, 0, 0.2);
    }

    .search-input::placeholder {
        color: #999;
    }

    .add-menu {
        position: absolute;
        top: 30px;
        left: 0;
        background: rgba(255, 255, 255, 1);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        border-radius: 5px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        padding: 0 0 1px 0;
        z-index: 11; /* 比 sticky-header 高 */
        min-width: 50px;
    }

    .add-menu button {
        display: block;
        width: 100%;
        padding: 6px 16px;
        background: none;
        border: none;
        text-align: left;
        color: #333;
        font-family: inherit;
        font-size: 14px;
        cursor: pointer;
    }

    .add-menu button:hover {
        background: rgba(0, 0, 0, 0.05);
    }

    /* 新增：弹窗样式 */
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.3);
        backdrop-filter: blur(2px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .dir-input-modal {
        background: rgba(255, 255, 255, 0.95);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        border-radius: 5px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
        padding: 8px;
        width: 200px;
        max-width: 90%;
    }

    .dir-input-modal input {
        width: 100%;
        padding: 2.5px 2.5px;
        border-radius: 3px;
        background: rgba(255, 255, 255, 0.8);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        border: 1px solid rgba(0, 0, 0, 0.15);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        font-family: inherit;
        font-size: 13px;
        color: #333;
        outline: none;
        transition: all 0.2s ease;
        margin-bottom: 6px;
        box-sizing: border-box;
    }

    .dir-input-modal input:focus {
        border-color: rgba(0, 0, 0, 0.25);
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
    }

    .modal-buttons {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
    }

    .modal-buttons button {
        padding: 5px 5px;
        border-radius: 4px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        font-family: inherit;
        font-size: 14px;
        cursor: pointer;
        transition: all 0.2s ease;
        min-width: 60px;
    }

    .cancel-btn {
        background: rgba(255, 255, 255, 0.8);
        color: #666;
    }

    .cancel-btn:hover {
        background: rgba(255, 255, 255, 1);
        color: #333;
    }

    .confirm-btn {
        background: rgba(37, 99, 235, 0.9);
        color: white;
        border-color: rgba(37, 99, 235, 0.3);
    }

    .confirm-btn:hover:not(:disabled) {
        background: rgba(37, 99, 235, 1);
        box-shadow: 0 4px 12px rgba(37, 99, 235, 0.2);
    }

    .confirm-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    /* App.svelte 中的样式 */
.context-menu {
  background: white;
  border: 1px solid #ddd;
  border-radius: 4px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  min-width: 150px;
  overflow: hidden;
  z-index: 9999;
}

.menu-list {
  list-style: none;
  margin: 0;
  padding: 4px 0;
}

.menu-item {
  padding: 6px 16px;
  cursor: pointer;
  font-size: 14px;
  color: #333;
  transition: background-color 0.2s;
  width: 100%;
  text-align: left;
}

.menu-item:hover {
  background-color: #f5f5f5;
}

.menu-item:last-child {
  color: #dc3545;
}

.menu-item:last-child:hover {
  background-color: #f8d7da;
  color: #721c24;
}
</style>