<script>
	import { slide } from "svelte/transition";
	import { LogInfo } from "../../wailsjs/runtime/runtime";

	// props
	export let itemKey; // 唯一标识符路径，例如 "0.test2.1"
	export let value; // 当前项的数据对象，例如 { "test1": "value" }
	export let data; // 根数据（用于在 Drop 时更新）
	export let updateData; // 更新根数据的回调
	export let expanded;
	export let toggleExpand;
	export let index; // [关键] 当前项在父数组中的索引

	let copied = false;
	let dragOverIndex = null;
	let isDragging = false;

	function copyToClipboard(text) {
		const content = typeof text === "string" ? text : JSON.stringify(text);
		navigator.clipboard
			.writeText(content)
			.then(() => {
				copied = true;
				setTimeout(() => (copied = false), 2000);
			})
			.catch((err) => {
				console.error("Failed to copy: ", err);
			});
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

		// 计算父级路径 (去掉最后一节就是父级路径)
		// 例如 "1.test2.3" -> 父路径 "1.test2"
		const pathParts = itemKey.split(".");
		pathParts.pop();
		const parentPath = pathParts.join(".");

		const dragInfo = { sourceParentPath: parentPath, sourceIndex: idx };
		e.dataTransfer.setData("application/json", JSON.stringify(dragInfo));
		e.dataTransfer.effectAllowed = "move";
	}

	function handleDragEnd(e) {
		e.stopPropagation();
		isDragging = false;
		dragOverIndex = null;
	}

	function handleDragOver(e, idx) {
		e.preventDefault(); // 允许 Drop
		e.stopPropagation();

		// 只有当实际上是同一层级时才显示蓝线（虽然Drop里也会校验，但视觉上最好也校验）
		// 这里简化处理，只设置状态，在 CSS 中配合
		if (dragOverIndex !== idx) {
			dragOverIndex = idx;
		}
	}

	function handleDragLeave(e) {
		// 关键判断：
		// e.relatedTarget 是鼠标进入的那个元素
		// e.currentTarget 是当前的 li
		// 如果鼠标进入的元素依然在当前 li 内部（比如碰到了内部的 span 或 button），
		// 则不认为是“离开”，不清除蓝线，防止闪烁。
		if (e.currentTarget.contains(e.relatedTarget)) return;

		dragOverIndex = null;
	}

	function stopContainerDrag(e) {
		// 防止拖拽到子列表缝隙时触发父级的 Drop
		e.preventDefault();
		e.stopPropagation();
	}

	function handleDrop(e, targetIndex) {
		e.preventDefault();
		e.stopPropagation();
		dragOverIndex = null;
		isDragging = false;

		try {
			const dragData = e.dataTransfer.getData("application/json");
			if (!dragData) return;

			const { sourceParentPath, sourceIndex } = JSON.parse(dragData);

			const pathParts = itemKey.split(".");
			pathParts.pop();
			const currentParentPath = pathParts.join(".");

			if (sourceParentPath !== currentParentPath) return;
			if (sourceIndex === targetIndex) return;

			let parentObj = data;
			if (currentParentPath !== "") {
				const keys = currentParentPath.split(".");
				for (let k of keys) {
					if (parentObj && parentObj[k] !== undefined) {
						parentObj = parentObj[k];
					}
				}
			}

			if (Array.isArray(parentObj)) {
				const newArray = [...parentObj];
				// 1. 先移除元素
				const [movedItem] = newArray.splice(sourceIndex, 1);

				// 2. 计算插入位置
				// 如果是从上往下拖（source < target），移除源元素后，目标位置的索引会减1
				let insertIndex = targetIndex;
				if (sourceIndex < targetIndex) {
					insertIndex -= 1;
				}

				// 3. 插入元素 (关键修复：这里要用 insertIndex，不能用 targetIndex)
				newArray.splice(insertIndex, 0, movedItem);

				// 4. 更新数据
				if (currentParentPath === "") {
					data = newArray;
					updateData(data);
				} else {
					updateNestedData(data, currentParentPath, newArray);
					// [核心修复]：根数据是数组，必须用 [...data]
					// 之前用的 {...data} 把数组变成了对象，导致了 {#each} 报错
					updateData([...data]);
				}
			}
		} catch (err) {
			console.error("Drop error", err);
		}
	}

	function updateNestedData(obj, path, newValue) {
		const keys = path.split(".");
		let current = obj;
		for (let i = 0; i < keys.length - 1; i++) {
			current = current[keys[i]];
		}
		current[keys[keys.length - 1]] = newValue;
	}


	export let showContextMenu;

	// 右键点击处理
	function handleContextMenu(e, key, val, isFolder) {
		showContextMenu(e, key, val, isFolder);
	}
</script>

<li
	class="tree-item"
	draggable="true"
	class:dragging={isDragging}
	class:drag-over-active={dragOverIndex === index}
	on:dragstart={(e) => handleDragStart(e, index)}
	on:dragend={handleDragEnd}
	on:dragover={(e) => handleDragOver(e, index)}
	on:dragleave={handleDragLeave}
	on:drop={(e) => handleDrop(e, index)}
>
	<!-- 针对你的数据结构：value 是一个对象 { key: content } -->
	{#each Object.entries(value) as [key, val] (key)}
		{#if Array.isArray(val)}
			<!-- 1. 值为数组 -> 渲染为文件夹 -->
			<button
				class="folder-btn"
				on:click={() => toggleExpand(itemKey + "." + key)}
				on:contextmenu={(e) =>
					handleContextMenu(e, itemKey + "." + key, val, true)}
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
				<!-- 绝缘层：on:dragover={stopContainerDrag} 阻止父级高亮 -->
				<ul
					class="nested-list"
					transition:slide={{ duration: 300 }}
					on:dragover={stopContainerDrag}
					on:drop={stopContainerDrag}
				>
					{#each val as subItem, subIndex (subIndex)}
						<!-- 递归渲染：注意 itemKey 要延续路径 -->
						<!-- [关键] 必须传递 index={subIndex} 供子组件计算排序位置 -->
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
			<!-- 2. 值为字符串 -> 渲染为普通行 -->
			<div
				class="item-line"
				on:click={() => copyToClipboard(val)}
				on:keydown={(e) => handleKeyCopy(e, val)}
				on:contextmenu={(e) =>
					handleContextMenu(e, itemKey + "." + key, val, false)}
				role="button"
				tabindex="0"
			>
				<span class="item-key">{key}</span>
				{#if copied}
					<span class="copied-indicator">已复制</span>
				{/if}
				<!-- 手柄 -->
				<span class="drag-handle" title="拖拽排序">⋮⋮</span>
			</div>
		{/if}
	{/each}
</li>

<style>
	/* --- 拖拽相关保持原样，功能性样式不动 --- */
	.tree-item.drag-over-active {
		border-top: 2px solid #3b82f6; /* 改用更现代的蓝色 */
		margin-top: -2px;
		z-index: 10;
		position: relative;
	}

	.tree-item.dragging {
		opacity: 0.4;
		background: #f0f0f0;
	}

	/* --- 容器 --- */
	.tree-item {
		margin: 0; /* 去掉外边距，让列表连续 */
		padding: 0;
		line-height: 1.4;
		list-style: none;
		user-select: none; /* 防止双击时不小心选中文字 */
	}

	/* --- 通用行样式 (文件夹按钮 和 文本行) --- */
	/* 核心改动：去掉白底阴影，改为透明底+Hover变色 */
	.folder-btn,
	.item-line {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 3px 8px; /* 极致压缩：上下3px */
		margin: 1px 0;    /* 极小间距 */
		background: transparent; /* 默认透明 */
		border: none;
		border-radius: 4px;      /* 微圆角 */
		cursor: pointer;
		font-size: 13px;         /* 配合主界面的小字体 */
		color: #333;
		text-align: left;
		transition: background-color 0.1s ease, color 0.1s;
		box-shadow: none; /* 去掉阴影 */
	}

	/* --- 鼠标悬停效果 --- */
	/* 像 VS Code 一样，悬停时给一个整行高亮 */
	.folder-btn:hover,
	.item-line:hover {
		background-color: rgba(0, 0, 0, 0.06); /* 浅灰背景 */
		transform: none; /* 去掉位移，防止列表抖动 */
		box-shadow: none;
		color: #000;
	}

	/* --- 文件夹特有样式 --- */
	.folder-btn {
		font-weight: 500; /* 文件夹稍微加粗一点点区分 */
		color: #444;
	}

	/* 图标微调 */
	.icon {
		margin-right: 6px;
		font-size: 14px; /* 图标也调小 */
		color: #666;     /* 灰色图标比黄色更高级 */
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;     /* 固定宽度对齐 */
	}

	.catalog-icon {
		width: 16px;
		height: 16px;
		margin-left: 4px;
		margin-right: 4px;
		vertical-align: middle;
		filter: brightness(0.8); /* 如果需要调整颜色深浅 */
	}

	/* --- 文本特有样式 --- */
	.label {
		flex: 1;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis; /* 文字过长显示省略号 */
		margin-right: 8px;
	}

	.item-key {
		color: #333;
		margin-right: 6px; /* 键值对之间的间距 */
		font-weight: 500;
	}
    
	/* --- 嵌套缩进 --- */
	.nested-list {
		margin-left: 10px; /* 缩进不需要太大 */
		padding-left: 10px;
		border-left: 1px solid rgba(0, 0, 0, 0.08); /* 极细的引导线 */
		list-style: none;
		margin-top: 0;
		margin-bottom: 0;
	}

	/* --- 拖拽手柄 --- */
	.drag-handle {
		margin-left: auto;
		padding: 0 4px;
		color: transparent; /* 默认隐藏，看起来更干净 */
		cursor: grab;
		font-size: 12px;
		display: flex;
		align-items: center;
	}

	/* 只有鼠标悬停在整行时，才显示拖拽手柄 */
	.folder-btn:hover .drag-handle,
	.item-line:hover .drag-handle {
		color: #bbb; /* 淡淡的灰色 */
	}
	
	.drag-handle:hover {
		color: #666 !important;
	}

	/* --- 复制成功提示 --- */
	.copied-indicator {
		margin-left: auto; /* 靠右显示 */
		padding-left: 8px;
		color: #10b981;    /* 绿色 */
		font-size: 11px;
		font-weight: 500;
		animation: fadeIn 0.2s ease;
        white-space: nowrap;
	}

	@keyframes fadeIn {
		from { opacity: 0; transform: translateX(5px); }
		to { opacity: 1; transform: translateX(0); }
	}
</style>