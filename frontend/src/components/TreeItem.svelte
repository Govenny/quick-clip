<script>
	import { slide } from "svelte/transition";
	// import { LogInfo } from "../../wailsjs/runtime/runtime"; // 暂时注释，防报错

	// props
	export let itemKey; 
	export let value; 
	export let data; 
	export let updateData; 
	export let expanded;
	export let toggleExpand;
	export let index; 

	let copied = false;
	let dragOverIndex = null; // 这里存储的是 index，用来高亮当前组件
	let isDragging = false;
	let dropType = null; // 'before', 'inside', 'after'

    // 监听：一旦 itemKey 或 value 发生变化（说明列表更新了），强制重置拖拽状态
    $: if (itemKey || value) {
        isDragging = false;
        dragOverIndex = null;
        dropType = null;
    }

	function copyToClipboard(text) {
		const content = typeof text === "string" ? text : JSON.stringify(text);
		navigator.clipboard.writeText(content).then(() => {
			copied = true;
			setTimeout(() => (copied = false), 2000);
		}).catch((err) => console.error("Failed to copy: ", err));
	}

	function handleKeyCopy(e, text) {
		if (e.key === "Enter" || e.key === " ") {
			e.preventDefault();
			copyToClipboard(text);
		}
	}

	// --- 核心拖拽逻辑 ---

	function handleDragStart(e, idx) {
		e.stopPropagation();
		isDragging = true;
		const dragInfo = { sourceKey: itemKey, sourceIndex: idx }; 
		e.dataTransfer.setData("application/json", JSON.stringify(dragInfo));
		e.dataTransfer.effectAllowed = "move";
	}

	function handleDragEnd(e) {
		e.stopPropagation();
		isDragging = false;
		dragOverIndex = null;
		dropType = null;
	}

	function handleDragOver(e, idx) {
		e.preventDefault();
		e.stopPropagation();

		// [核心修复] currentTarget 现在是 button 或 div 行，高度固定且准确
		const rect = e.currentTarget.getBoundingClientRect();
		const relativeY = e.clientY - rect.top;
		const height = rect.height;

		// 分区域判定 (更灵敏的参数)
		// 上 25% -> Before
		// 下 25% -> After
		// 中间 50% -> Folder ? Inside : After
		if (relativeY < height * 0.25) {
			dropType = 'before';
		} else if (relativeY > height * 0.75) {
			dropType = 'after';
		} else {
			const [key, val] = Object.entries(value)[0];
			if (Array.isArray(val)) {
				dropType = 'inside';
			} else {
				dropType = 'after'; // 普通行中间区域也视为排序（插在后面）
			}
		}
		dragOverIndex = idx;
	}

	function handleDrop(e, targetIndex) {
		e.preventDefault();
		e.stopPropagation();
		
		const dragDataStr = e.dataTransfer.getData("application/json");
		if (!dragDataStr) return;
		const dragData = JSON.parse(dragDataStr);
		const sourceKey = dragData.sourceKey;

		// 缓存当前状态，因为 reset 后会被清空
		const currentDropType = dropType;
		
		// 重置状态
		dragOverIndex = null;
		dropType = null;
		isDragging = false;

		// 获取当前项父路径
		const pathParts = itemKey.split(".");
		pathParts.pop();
		const currentParentPath = pathParts.join(".");
		
		const [key, val] = Object.entries(value)[0];

		if (currentDropType === 'inside') {
			// 移入文件夹
			const targetFolderPath = itemKey + "." + key;
			// 防止自己拖进自己
			if (sourceKey === targetFolderPath || targetFolderPath.startsWith(sourceKey + ".")) return;
			moveItem(sourceKey, targetFolderPath, -1);
		} else {
			// 排序
			let finalIndex = targetIndex;
			if (currentDropType === 'after') finalIndex += 1;
			moveItem(sourceKey, currentParentPath, finalIndex);
		}
	}

	function moveItem(srcKey, destParentPath, destIdx) {
		let newData = JSON.parse(JSON.stringify(data));

		// 1. 移除源
		let movedItem = null;
		const srcParts = srcKey.split(".");
		const srcIdx = parseInt(srcParts.pop());
		const srcParentPath = srcParts.join(".");

		let srcParent = getByPath(newData, srcParentPath);
		// 兼容处理：如果路径指向的是对象 key:value 结构
		if (srcParent && !Array.isArray(srcParent) && typeof srcParent === 'object') {
             // 这一步根据你的数据结构可能不需要，但为了保险
             const k = srcParentPath.split(".").pop();
             if (srcParent[k]) srcParent = srcParent[k];
        }

		if (Array.isArray(srcParent)) {
			[movedItem] = srcParent.splice(srcIdx, 1);
		} else {
            console.error("Source parent not array", srcParentPath, srcParent);
            return;
        }

		// 2. 插入目标
		let destParent = getByPath(newData, destParentPath);
		
		// 特殊处理：如果路径指向的是文件夹对象 { "Folder": [] }
		if (destParent && !Array.isArray(destParent)) {
			// 尝试获取最后一段 key，看看是不是在这个对象里
			const folderKey = destParentPath.split(".").pop();
			if (destParent[folderKey] && Array.isArray(destParent[folderKey])) {
				destParent = destParent[folderKey];
			} else {
                // 如果 split 不对，可能是 drop 传入的路径已经是准确的父级对象
                // 这种情况下通常 destParent 本身应该是包含数组的父级
                // 你的数据结构似乎是 [ {key: val}, {key: []} ]
                // 这里需要根据 getByPath 的返回值做防御
            }
		}

		if (Array.isArray(destParent)) {
			if (destIdx === -1) destParent.push(movedItem);
			else destParent.splice(destIdx, 0, movedItem);
			// [关键修改] 使用 setTimeout 将数据更新推迟到下一个事件循环
			// 这样可以让浏览器的 dragend 事件先执行完毕，重置样式
			setTimeout(() => {
				updateData(newData);
			}, 0);
		} else {
            console.error("Dest parent not array", destParentPath, destParent);
        }
	}

	function getByPath(obj, path) {
		if (path === "") return obj;
		const parts = path.split(".");
        let curr = obj;
        for (const p of parts) {
            if (curr && curr[p] !== undefined) {
                curr = curr[p];
            } else {
                // 针对你的结构：数组项是对象 {key: val}
                // 路径可能是 "0.Folder.1"
                // 0 -> arr[0] -> {Folder: []}
                // .Folder -> []
                // .1 -> item
                // 如果 get 失败，可能是因为遇到对象包裹
                // 但通常 split 逻辑应该匹配数据结构
                return undefined;
            }
        }
        return curr;
	}

	export let showContextMenu;
	function handleContextMenu(e, key, val, isFolder) {
		showContextMenu(e, key, val, isFolder);
	}
</script>

<!-- 
    [修改点 1] li 不再负责拖拽事件和样式 
    它只作为结构容器，这样可以避免高度计算错误
-->
<li class="tree-item">
	{#each Object.entries(value) as [key, val] (key)}
		{#if Array.isArray(val)}
			<!-- 
                [修改点 2] 拖拽逻辑全部移到这个 button 上 
                因为它是文件夹的“标题行”，高度固定 (~30px)
            -->
			<button
				class="folder-btn"
                draggable="true"
                class:dragging={isDragging}
                class:drop-before={dragOverIndex === index && dropType === 'before'}
                class:drop-after={dragOverIndex === index && dropType === 'after'}
                class:drop-inside={dragOverIndex === index && dropType === 'inside'}
				on:click={() => toggleExpand(itemKey + "." + key)}
                on:dragstart={(e) => handleDragStart(e, index)}
                on:dragover={(e) => handleDragOver(e, index)}
                on:dragleave={() => { dragOverIndex = null; dropType = null; }}
                on:dragend={handleDragEnd}
                on:drop={(e) => handleDrop(e, index)}
				on:contextmenu={(e) => handleContextMenu(e, itemKey + "." + key, val, true)}
			>
				<span class="icon">
					<img
						class="catalog-icon"
						src={expanded[itemKey + "." + key] ? "/src/assets/images/catalog-expand.png" : "/src/assets/images/catalog.png"}
						alt={expanded[itemKey + "." + key] ? "收起" : "展开"}
					/>
				</span>
				<span class="label">{key}</span>
				<span class="drag-handle" title="拖拽排序">⋮⋮</span>
			</button>

			{#if expanded[itemKey + "." + key]}
				<ul
					class="nested-list"
					transition:slide={{ duration: 200 }}
				>
					{#each val as subItem, subIndex (subIndex)}
						<svelte:self
							itemKey={itemKey + "." + key + "." + subIndex}
							value={subItem}
							{data}
							{updateData}
							{expanded}
							{toggleExpand}
							index={subIndex}
							showContextMenu={showContextMenu}
						/>
					{/each}
				</ul>
			{/if}
		{:else}
			<!-- 
                [修改点 3] 普通文本行同理，添加完整的拖拽属性
            -->
			<div
				class="item-line"
                draggable="true"
                class:dragging={isDragging}
                class:drop-before={dragOverIndex === index && dropType === 'before'}
                class:drop-after={dragOverIndex === index && dropType === 'after'}
                on:dragstart={(e) => handleDragStart(e, index)}
                on:dragover={(e) => handleDragOver(e, index)}
                on:dragleave={() => { dragOverIndex = null; dropType = null; }}
                on:dragend={handleDragEnd}
                on:drop={(e) => handleDrop(e, index)}
				on:click={() => copyToClipboard(val)}
				on:keydown={(e) => handleKeyCopy(e, val)}
				on:contextmenu={(e) => handleContextMenu(e, itemKey + "." + key, val, false)}
				role="button"
				tabindex="0"
			>
				<span class="item-key">{key}</span>
				{#if copied}
					<span class="copied-indicator">已复制</span>
				{/if}
				<span class="drag-handle" title="拖拽排序">⋮⋮</span>
			</div>
		{/if}
	{/each}
</li>

<style>
	/* 容器去掉 padding margin，只作为 wrapper */
	.tree-item {
		margin: 0;
		padding: 0;
		list-style: none;
	}

    /* 
       [修改点 4] 样式全部针对 folder-btn 和 item-line 
       注意：position: relative 是必须的，为了定位蓝线 
    */
    .folder-btn,
    .item-line {
        position: relative; /* 关键 */
        /* ... 其他原有样式保持不变 ... */
        display: flex;
        align-items: center;
        width: 100%;
        padding: 3px 8px;
        margin: 1px 0;
        background: transparent;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-size: 13px;
        color: #333;
        text-align: left;
        transition: background-color 0.1s ease;
    }

	/* 拖拽时的半透明 */
	.dragging {
		opacity: 0.4;
        background: #f5f5f5;
	}

	/* 移入文件夹高亮 */
	.folder-btn.drop-inside {
		background-color: rgba(59, 130, 246, 0.2) !important;
        color: #000;
	}

	/* 排序指示线：上方 */
	.folder-btn.drop-before::before,
    .item-line.drop-before::before {
		content: "";
		position: absolute;
		top: -2px; /* 往外一点点，更清晰 */
		left: 0;
		right: 0;
		height: 2px;
		background: #3b82f6;
		z-index: 10;
        pointer-events: none;
	}

	/* 排序指示线：下方 */
	.folder-btn.drop-after::after,
    .item-line.drop-after::after {
		content: "";
		position: absolute;
		bottom: -2px;
		left: 0;
		right: 0;
		height: 2px;
		background: #3b82f6;
		z-index: 10;
        pointer-events: none;
	}

	/* Hover 效果 */
	.folder-btn:hover,
	.item-line:hover {
		background-color: rgba(0, 0, 0, 0.06);
	}

    /* ... 其他图标、文字、复制提示样式保持不变 ... */
	.folder-btn { font-weight: 500; color: #444; }
	.icon { margin-right: 6px; display: flex; align-items: center; width: 16px; }
	.catalog-icon { width: 16px; height: 16px; }
	.label { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-right: 8px; }
	.item-key { color: #333; margin-right: 6px; font-weight: 500; }
	.nested-list { margin-left: 10px; padding-left: 10px; border-left: 1px solid rgba(0, 0, 0, 0.08); list-style: none; }
	.drag-handle { margin-left: auto; color: transparent; cursor: grab; font-size: 12px; }
	.folder-btn:hover .drag-handle, .item-line:hover .drag-handle { color: #bbb; }
	.drag-handle:hover { color: #666 !important; }
	.copied-indicator { margin-left: auto; padding-left: 8px; color: #10b981; font-size: 11px; animation: fadeIn 0.2s ease; }
	@keyframes fadeIn { from { opacity: 0; transform: translateX(5px); } to { opacity: 1; transform: translateX(0); } }
</style>