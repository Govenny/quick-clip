<script>
	import { slide } from "svelte/transition";
	import { quartOut } from 'svelte/easing';
	import { PasteAndHide, HideAndRestore, RecordItemUsage } from "../../wailsjs/go/main/App";

	// props: 规范的树节点对象与全局操作回调
	export let node;
	export let expanded;
	export let toggleExpand;
	export let showContextMenu;
	export let onMoveNode;
	export let autoPaste = true;

	let copied = false;
	let dragOverThis = false;
	let isDragging = false;
	let dropType = null; // 'before', 'inside', 'after'

	// 一旦 node 变化，重置状态
	$: if (node) {
		isDragging = false;
		dragOverThis = false;
		dropType = null;
	}

	function copyToClipboard(text) {
		if (node && node.id) {
			RecordItemUsage(node.id);
		}
		const content = typeof text === "string" ? text : JSON.stringify(text ?? "");
		navigator.clipboard.writeText(content).then(() => {
			copied = true;
			setTimeout(() => (copied = false), 2000);
		}).catch((err) => console.error("Failed to copy: ", err));

		if (autoPaste) {
			PasteAndHide();
		} else {
			HideAndRestore();
		}
	}

	function handleKeyCopy(e, text) {
		if (e.key === "Enter" || e.key === " ") {
			e.preventDefault();
			copyToClipboard(text);
		}
	}

	// --- 核心拖拽逻辑 (基于唯一 node.id) ---

	function handleDragStart(e) {
		e.stopPropagation();
		isDragging = true;
		const dragInfo = { sourceId: node.id };
		e.dataTransfer.setData("application/json", JSON.stringify(dragInfo));
		e.dataTransfer.effectAllowed = "move";
	}

	function handleDragEnd(e) {
		e.stopPropagation();
		isDragging = false;
		dragOverThis = false;
		dropType = null;
	}

	function handleDragOver(e) {
		e.preventDefault();
		e.stopPropagation();

		const rect = e.currentTarget.getBoundingClientRect();
		const relativeY = e.clientY - rect.top;
		const height = rect.height;

		if (node.type === 'folder') {
			// 上 25% -> Before, 下 25% -> After, 中间 50% -> Inside
			if (relativeY < height * 0.25) {
				dropType = 'before';
			} else if (relativeY > height * 0.75) {
				dropType = 'after';
			} else {
				dropType = 'inside';
			}
		} else {
			// 普通文本只有 before 和 after
			if (relativeY < height * 0.5) {
				dropType = 'before';
			} else {
				dropType = 'after';
			}
		}
		dragOverThis = true;
	}

	function handleDrop(e) {
		e.preventDefault();
		e.stopPropagation();

		const dragDataStr = e.dataTransfer.getData("application/json");
		if (!dragDataStr) return;

		let sourceId;
		try {
			const dragData = JSON.parse(dragDataStr);
			sourceId = dragData.sourceId;
		} catch (err) {
			return;
		}

		const currentDropType = dropType;

		dragOverThis = false;
		dropType = null;
		isDragging = false;

		if (sourceId && sourceId !== node.id && currentDropType && onMoveNode) {
			onMoveNode(sourceId, node.id, currentDropType);
		}
	}

	function handleContextMenu(e) {
		e.preventDefault();
		e.stopPropagation();
		if (showContextMenu) {
			showContextMenu(e, node);
		}
	}
</script>

<li class="tree-item">
	{#if node.type === 'folder'}
		<!-- 文件夹节点行 -->
		<button
			class="folder-btn"
			draggable="true"
			class:dragging={isDragging}
			class:drop-before={dragOverThis && dropType === 'before'}
			class:drop-after={dragOverThis && dropType === 'after'}
			class:drop-inside={dragOverThis && dropType === 'inside'}
			on:click={() => toggleExpand(node.id)}
			on:dragstart={handleDragStart}
			on:dragover={handleDragOver}
			on:dragleave={() => { dragOverThis = false; dropType = null; }}
			on:dragend={handleDragEnd}
			on:drop={handleDrop}
			on:contextmenu={handleContextMenu}
		>
			<span class="folder-icon" aria-hidden="true">
				<svg class="chevron" class:expanded={expanded[node.id]} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="9 18 15 12 9 6"></polyline>
				</svg>
				{#if expanded[node.id]}
					<svg class="folder-svg open" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
						<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
						<polygon points="2 10 22 10 19 21 5 21"></polygon>
					</svg>
				{:else}
					<svg class="folder-svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
						<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
					</svg>
				{/if}
			</span>
			<span class="label" title={node.name}>{node.name}</span>
			<span class="drag-handle" title="拖拽排序">⋮⋮</span>
		</button>

		{#if expanded[node.id] && node.children && node.children.length > 0}
			<ul
				class="nested-list"
				transition:slide={{ duration: 280, easing: quartOut }}
			>
				{#each node.children as subNode (subNode.id)}
					<svelte:self
						node={subNode}
						{expanded}
						{toggleExpand}
						{showContextMenu}
						{onMoveNode}
						{autoPaste}
					/>
				{/each}
			</ul>
		{/if}
	{:else}
		<!-- 文本节点行 -->
		<div
			class="item-line"
			draggable="true"
			class:dragging={isDragging}
			class:drop-before={dragOverThis && dropType === 'before'}
			class:drop-after={dragOverThis && dropType === 'after'}
			on:dragstart={handleDragStart}
			on:dragover={handleDragOver}
			on:dragleave={() => { dragOverThis = false; dropType = null; }}
			on:dragend={handleDragEnd}
			on:drop={handleDrop}
			on:click={() => copyToClipboard(node.value)}
			on:keydown={(e) => handleKeyCopy(e, node.value)}
			on:contextmenu={handleContextMenu}
			role="button"
			tabindex="0"
			title={typeof node.value === "string" ? node.value : ""}
		>
			<span class="item-key" title={node.name}>{node.name}</span>
			{#if typeof node.value === "string" && node.value.includes("\n")}
				<span class="multiline-indicator" title={`Multiline content (${node.value.split("\n").length} lines)`}>
					{node.value.split("\n").length} lines
				</span>
			{/if}
			{#if copied}
				<span class="copied-indicator">已复制</span>
			{/if}
			<span class="drag-handle" title="拖拽排序">⋮⋮</span>
		</div>
	{/if}
</li>

<style>
	.tree-item {
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.folder-btn,
	.item-line {
		position: relative;
		display: flex;
		align-items: center;
		width: 100%;
		min-width: 0;
		max-width: 100%;
		box-sizing: border-box;
		padding: 3px 8px;
		margin: 1px 0;
		background: transparent;
		border: none;
		border-radius: 4px;
		cursor: pointer;
		min-height: var(--app-item-line-height, 28px);
		font-size: var(--app-font-size, 13px);
		color: #333;
		text-align: left;
		box-shadow: inset 0 -1px 0 rgba(0, 0, 0, 0.045);
		transition: background-color 0.18s ease, box-shadow 0.18s ease, color 0.18s ease;
	}

	.dragging {
		opacity: 0.4;
		background: #f5f5f5;
		transition: opacity 0.3s cubic-bezier(0.34, 1.3, 0.64, 1), background-color 0.25s cubic-bezier(0.34, 1.3, 0.64, 1);
	}

	.folder-btn.drop-inside {
		background-color: rgba(71, 94, 117, 0.15) !important;
		color: #1e293b;
		transition: background-color 0.3s cubic-bezier(0.34, 1.56, 0.64, 1), color 0.25s cubic-bezier(0.34, 1.3, 0.64, 1);
	}

	.folder-btn.drop-before::before,
	.item-line.drop-before::before {
		content: "";
		position: absolute;
		top: -2px;
		left: 0;
		right: 0;
		height: 2px;
		background: #5b7a9c;
		z-index: 10;
		pointer-events: none;
	}

	.folder-btn.drop-after::after,
	.item-line.drop-after::after {
		content: "";
		position: absolute;
		bottom: -2px;
		left: 0;
		right: 0;
		height: 2px;
		background: #5b7a9c;
		z-index: 10;
		pointer-events: none;
	}

	.folder-btn:hover,
	.item-line:hover {
		background-color: rgba(232, 239, 247, 0.75);
		box-shadow:
			inset 0 0 0 1px rgba(255, 255, 255, 0.9),
			inset 0 -1px 0 rgba(91, 108, 128, 0.08),
			0 2px 6px rgba(31, 41, 55, 0.06);
		color: #1f2937;
	}

	.folder-btn {
		font-weight: 500;
		color: #334155;
		background: rgba(241, 245, 249, 0.32);
		box-shadow:
			inset 0 1px 0 rgba(255, 255, 255, 0.65),
			inset 0 -1px 0 rgba(71, 85, 105, 0.06);
	}
	.folder-btn:hover {
		background-color: rgba(228, 236, 245, 0.65);
		box-shadow:
			inset 0 0 0 1px rgba(255, 255, 255, 0.88),
			inset 0 -1px 0 rgba(71, 85, 105, 0.1),
			0 2px 8px rgba(30, 41, 59, 0.06);
	}
	.folder-icon {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		margin-right: 6px;
		flex-shrink: 0;
	}
	.chevron {
		width: 12px;
		height: 12px;
		color: #8c9dae;
		flex-shrink: 0;
		transform-origin: center;
		transition: transform 0.2s cubic-bezier(0.34, 1.3, 0.64, 1), color 0.18s ease;
	}
	.chevron.expanded {
		transform: rotate(90deg);
		color: #475e75;
	}
	.folder-svg {
		width: 15px;
		height: 15px;
		color: #5a6f84;
		fill: rgba(90, 111, 132, 0.08);
		flex-shrink: 0;
		transition: color 0.18s ease, fill 0.18s ease;
	}
	.folder-svg.open {
		color: #475e75;
		fill: rgba(71, 94, 117, 0.14);
	}
	.folder-btn:hover .chevron {
		color: #3a5168;
	}
	.folder-btn:hover .folder-svg {
		color: #3a5168;
		fill: rgba(71, 94, 117, 0.18);
	}
	.label {
		flex: 1;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		margin-right: 8px;
		color: #334155;
		font-weight: 550;
		letter-spacing: 0;
		line-height: 1.4;
	}
	.item-key {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: #2f3b47;
		margin-right: 6px;
		font-weight: 500;
		letter-spacing: 0;
	}
	.multiline-indicator {
		flex: 0 0 auto;
		margin-right: 6px;
		padding: 1px 5px;
		border: 1px solid rgba(148, 163, 184, 0.22);
		border-radius: 3px;
		background: rgba(226, 232, 240, 0.5);
		color: #475569;
		font-size: 10px;
		font-weight: 500;
		line-height: 14px;
		white-space: nowrap;
	}
	.nested-list {
		width: calc(100% - 10px);
		max-width: calc(100% - 10px);
		margin-left: 10px;
		padding-left: 10px;
		box-sizing: border-box;
		border-left: 1px solid rgba(0, 0, 0, 0.08);
		list-style: none;
	}
	.drag-handle {
		flex: 0 0 auto;
		margin-left: auto;
		color: transparent;
		cursor: grab;
		font-size: 12px;
		transition: color 0.25s cubic-bezier(0.34, 1.3, 0.64, 1);
	}
	.folder-btn:hover .drag-handle, .item-line:hover .drag-handle { color: #bbb; }
	.drag-handle:hover { color: #666 !important; }
	.copied-indicator {
		margin-left: auto;
		padding-left: 8px;
		color: #10b981;
		font-size: 11px;
		animation: fadeIn 0.3s cubic-bezier(0.34, 1.3, 0.64, 1);
	}
	@keyframes fadeIn {
		from { opacity: 0; transform: translateX(6px); }
		to { opacity: 1; transform: translateX(0); }
	}
</style>