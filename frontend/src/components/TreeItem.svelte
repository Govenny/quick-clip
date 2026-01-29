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
				<span class="icon"
					>{expanded[itemKey + "." + key] ? "📂" : "📁"}</span
				>
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
	/* --- 样式部分保持一致 --- */
	.tree-item.drag-over-active {
		border-top: 2px solid #007bff;
		margin-top: -2px;
		z-index: 10;
		position: relative;
	}

	.tree-item.dragging {
		opacity: 0.4;
	}

	.drag-handle {
		margin-left: auto;
		padding: 2px 8px;
		color: #ddd;
		cursor: grab;
		font-size: 16px;
		font-weight: bold;
		line-height: 1;
		user-select: none;
		transition: color 0.2s;
	}

	.folder-btn:hover .drag-handle,
	.item-line:hover .drag-handle {
		color: #888;
	}

	.tree-item {
		margin: 1.5px 0;
		transition: all 0.2s ease;
		line-height: 1.3;
		list-style: none;
	}

	.folder-btn {
		background: rgba(255, 255, 255, 0.9);
		border: none;
		border-radius: 4px;
		cursor: pointer;
		font-size: 14px;
		display: flex;
		align-items: center;
		padding: 6px 12px;
		width: 100%;
		text-align: left;
		transition: all 0.3s ease;
		box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
		color: #333;
		font-weight: normal;
	}
	.folder-btn:hover {
		background: rgba(255, 255, 255, 1);
		transform: translateY(-1px);
		box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
	}

	.icon {
		margin-right: 8px;
		font-size: 16px;
		transition: transform 0.3s ease;
	}
	.label {
		flex: 1;
		font-weight: normal;
		color: #333;
	}

	.nested-list {
		margin-left: 2px;
		padding-left: 2px;
		border-left: 1px solid rgba(0, 0, 0, 0.1);
		list-style: none;
		margin-top: 2px;
	}

	.item-key {
		font-weight: normal;
		color: #333;
		font-size: 14px;
	}

	.item-line {
		display: flex;
		align-items: center;
		padding: 6px 12px;
		background: rgba(255, 255, 255, 0.7);
		backdrop-filter: blur(5px);
		border-radius: 4px;
		margin: 1.5px 0;
		transition: all 0.3s ease;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
		font-size: 14px;
		line-height: 1.3;
		cursor: pointer;
	}
	.item-line:hover {
		background: rgba(255, 255, 255, 0.9);
		transform: translateY(-1px);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
	}

	.copied-indicator {
		margin-left: 8px;
		color: #28a745;
		font-size: 12px;
		font-weight: bold;
		animation: fadeIn 0.3s ease;
	}
	@keyframes fadeIn {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}
</style>
