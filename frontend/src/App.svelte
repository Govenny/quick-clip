<script>
    import { onMount, tick } from 'svelte';
    import { fade, fly} from 'svelte/transition';
    // 注意：请确认 wailsjs 的路径是否正确，根据你的项目结构调整
    import { GetContent, SaveContent } from '../wailsjs/go/main/App'; 
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

    onMount(async () => {
        try {
            data = await GetContent();
        } catch (error) {
            console.error('Failed to load content:', error);
        }
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
            event.preventDefault(); // 阻止默认提交行为

            if (isTitleInput) {
                // 标题输入框：回车→聚焦到值输入框
                textInputRef?.focus();
            } else {
                // 值输入框：回车→如果表单有效则确认
                if (isFormValid) {
                    confirmAddText();
                }
            }
        } else if (key === 'Escape') {
            // ESC键：取消
            cancelAddText();
        } else if (key === 'Tab' && isTitleInput) {
            // 标题输入框按Tab：跳到值输入框
            if (event.shiftKey === false) {
                event.preventDefault();
                textInputRef?.focus();
            }
        }
    }

    function confirmAddText() {
        if (!isFormValid) {
            // 可以显示错误提示
            if (!titleName.trim()) {
                titleInputRef?.focus();
                alert("请输入名称");
            } else if (!textName) {
                textInputRef?.focus();
                alert("请输入值");
            }
            return;
        }
        const newDir = { [titleName.trim()]: textName };
        data = [...data, newDir];
        updateData(data);

        // 清理并关闭
        titleName = "";
        textName = "";
        showTextInput = false;
    }

    function cancelAddText() {
        titleName = "";
        textName = "";
        showTextInput = false;
    }

    // 自动聚焦
    $: if (showTextInput && titleInputRef) {
        setTimeout(() => titleInputRef.focus(), 0);
    }

    function addDir() {
        // 显示目录名称输入弹窗
        showDirInput = true;
        dirName = ""; // 清空输入框
        showMenu = false; // 关闭菜单

        tick().then(() => {
            if (dirInputRef) {
                dirInputRef.focus();
            }
        });
    }

    function confirmAddDir() {
        const dirNameTrim = dirName.trim()
        if (dirName.trim() !== "") {
            // 创建一个新的目录对象，名称为用户输入，值为空数组
            const newDir = { [dirNameTrim]: [] };
            data = [...data, newDir];
            updateData(data);
        }
        // 关闭弹窗
        showDirInput = false;
        dirName = "";
    }

    function cancelAddDir() {
        showDirInput = false;
        dirName = "";
    }

    // 更新函数，TreeItem 会调用此函数更新 data
    function updateData(newData) {
        data = newData;
        SaveContent(data)
        // 如果需要持久化到后端，可以在这里调用 SaveContent(data) 等方法
    }
</script>

<h3 class="main-title" 
    on:mouseenter={() => isHovered = true} 
    on:mouseleave={() => isHovered = false}
    class:has-hover={isHovered}
    style="pointer-events: auto;">
    Quick-Clip
</h3>
<div class="app-container" class:has-hover={isHovered}>
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
    {#if data.length === 0}
        <p class="loading">Loading...</p>
    {:else}
        <ul class="content-list">
            {#each data as item, index (index)}
                <!-- 
                     重要修改：
                     1. 传递 index={index}，让组件知道自己在数组中的位置。
                     2. 添加 (index) 作为 each 的 key，或者最好使用 unique ID。
                        如果没有 ID，使用 index 配合拖拽可能会有轻微副作用，但对于简单数组重排通常是有效的。
                        为了让动画和状态保持更佳，如果你的数据有唯一ID最好用 `(item.id)`。
                        既然这里没有ID，暂时用 `(item)` 或者 `(index)`。
                -->
                <TreeItem 
                    itemKey={index.toString()} 
                    value={item} 
                    {data} 
                    {updateData} 
                    {expanded} 
                    {toggleExpand} 
                    index={index} 
                />
            {/each}
        </ul>
    {/if}
</div>

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
        padding: 1.5% 2% 30px;
        width: 600px;
        max-width: 600px;
        text-align: left;
        transition: all 0.3s ease;
        overflow: auto;
        margin: 0 auto;
        position: relative;
        z-index: 2;
        /* 建议给一个最大高度，不然无限长 */
        max-height: 80vh;
        transform: translateY(0);
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
    }

    .content-list {
        list-style: none;
        padding: 0;
        margin: 0;
        /* 给列表本身加一点 padding 防止拖拽时边界判定问题 */
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
        margin-bottom: 5px
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
        background: rgba(255, 255, 255, 0.9);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        border-radius: 5px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        padding: 0 0 1px 0;
        z-index: 10;
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
</style>