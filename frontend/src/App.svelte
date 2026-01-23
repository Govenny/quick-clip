<script>
    import { onMount } from 'svelte';
    import { fade } from 'svelte/transition';
    // 注意：请确认 wailsjs 的路径是否正确，根据你的项目结构调整
    import { GetContent, SaveContent } from '../wailsjs/go/main/App'; 
    import TreeItem from './components/TreeItem.svelte';

    let data = [];
    let expanded = {};
    let showMenu = false;
    let isHovered = false;

    onMount(async () => {
        try {
            data = await GetContent();
        } catch (error) {
            console.error('Failed to load content:', error);
            // 测试用数据
            /*
            data = [
                { "test1": "test1" },
                { "test2": [ { "test21": "test21" }, { "test22": "test22" } ] },
                { "test3": "test3" }
            ];
            */
        }
    });

    function toggleExpand(key) {
        expanded[key] = !expanded[key];
    }

    function toggleMenu() {
        showMenu = !showMenu;
    }

    function addText() {
        data = [...data, "新文本"];
        updateData(data);
        showMenu = false;
    }

    function addDir() {
        data = [...data, { "新目录": [] }];
        updateData(data);
        showMenu = false;
    }

    // 更新函数，TreeItem 会调用此函数更新 data
    function updateData(newData) {
        data = newData;
        SaveContent(data)
        // 如果需要持久化到后端，可以在这里调用 SaveContent(data) 等方法
    }
</script>

<h3 class="main-title" on:mouseenter={() => isHovered = true} on:mouseleave={() => isHovered = false}>Quick-Clip</h3>
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

<style>
    .app-container {
        background: rgba(255, 255, 255, 0.8);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        border-radius: 6px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        padding: 40px 2% 30px;
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
        transform: translateX(-50%);
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
        pointer-events: none;
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
        transition: background 0.2s;
    }

    .add-menu button:hover {
        background: rgba(0, 0, 0, 0.05);
    }
</style>