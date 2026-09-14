<script>
    import { createEventDispatcher } from 'svelte';
    import { scale, fade } from 'svelte/transition';

    export let visible = false;
    export let x = 0;
    export let y = 0;
    export let flipX = false;
    export let flipY = false;
    export let targetNode = null;
    export let isCapsule = false;
    export let capsuleSlot = null; // { slot: 1|2|3, type: 'pinned'|'auto', item: node }
    export let slots = []; // 3 个槽位对象
    export let capsuleShortcuts = [["Alt", "1"], ["Alt", "2"], ["Alt", "3"]];

    const dispatch = createEventDispatcher();

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

    function getShortcutLabel(idx) {
        const sc = capsuleShortcuts[idx];
        if (!sc || !Array.isArray(sc)) return `Alt+${idx + 1}`;
        const [mod, key] = sc;
        if (!mod || mod === 'None') return key;
        return `${mod}+${key}`;
    }

    function getSlotSummary(s) {
        if (!s || s.type === 'empty' || !s.item) {
            return '空';
        }
        const name = s.item.name || '';
        return name.length > 7 ? name.slice(0, 6) + '…' : name;
    }
</script>

{#if visible}
    <div 
        class="context-menu"
        style="position: fixed; top: {y}px; left: {x}px; transform-origin: {flipX ? 'right' : 'left'} {flipY ? 'bottom' : 'top'};"
        in:scale={{ duration: 130, easing: iosElastic }} 
        out:fade={{ duration: 60 }}
        on:contextmenu|preventDefault
    >
        {#if isCapsule}
            {#if capsuleSlot?.type === 'pinned'}
                <div class="menu-item" on:click={() => dispatch('unpinSlot', capsuleSlot.slot)} on:keydown={(e) => e.key === 'Enter' && dispatch('unpinSlot', capsuleSlot.slot)}>
                    <svg class="item-icon-svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="2" y1="2" x2="22" y2="22"></line>
                        <line x1="12" y1="17" x2="12" y2="22"></line>
                        <path d="M5 17h14v-2l-2-2V5h1V3H6v2h1v8l-2 2v2z"></path>
                    </svg>
                    <span>取消固定 (释放槽位)</span>
                </div>
            {:else}
                <div class="menu-item" on:click={() => dispatch('pinHere', capsuleSlot)} on:keydown={(e) => e.key === 'Enter' && dispatch('pinHere', capsuleSlot)}>
                    <svg class="item-icon-svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="12" y1="17" x2="12" y2="22"></line>
                        <path d="M5 17h14v-2l-2-2V5h1V3H6v2h1v8l-2 2v2z"></path>
                    </svg>
                    <span>固定在此槽位</span>
                </div>
                <div class="menu-item" on:click={() => dispatch('removeCapsule')} on:keydown={(e) => e.key === 'Enter' && dispatch('removeCapsule')}>
                    <span>从常用移除</span>
                </div>
            {/if}
            <div class="menu-divider"></div>
            <div class="menu-item" on:click={() => dispatch('editText')} on:keydown={(e) => e.key === 'Enter' && dispatch('editText')}>Edit</div>
            <div class="menu-divider"></div>
            <div class="menu-item delete" on:click={() => dispatch('delete')} on:keydown={(e) => e.key === 'Enter' && dispatch('delete')}>Delete</div>
        {:else}
            {#if targetNode?.type === 'folder'}
                <div class="menu-item" on:click={() => dispatch('addText')} on:keydown={(e) => e.key === 'Enter' && dispatch('addText')}>New Text</div>
                <div class="menu-item" on:click={() => dispatch('addDir')} on:keydown={(e) => e.key === 'Enter' && dispatch('addDir')}>New Folder</div>
                <div class="menu-divider"></div>
                <div class="menu-item" on:click={() => dispatch('editDir')} on:keydown={(e) => e.key === 'Enter' && dispatch('editDir')}>Edit</div>
                <div class="menu-divider"></div>
            {/if}
            {#if targetNode?.type === 'text'}
                <div class="menu-item submenu-trigger">
                    <span class="menu-label">
                        <svg class="item-icon-svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                            <line x1="12" y1="17" x2="12" y2="22"></line>
                            <path d="M5 17h14v-2l-2-2V5h1V3H6v2h1v8l-2 2v2z"></path>
                        </svg>
                        固定至常用槽位
                    </span>
                    <svg class="chevron-right" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <polyline points="9 18 15 12 9 6"></polyline>
                    </svg>

                    <div class="submenu" class:flip-left={flipX} class:flip-up={flipY}>
                        {#each [1, 2, 3] as slotNum, idx}
                            {@const currSlot = slots.find(s => s.slot === slotNum)}
                            <div 
                                class="menu-item submenu-item" 
                                on:click|stopPropagation={() => dispatch('assignSlot', { slot: slotNum, targetNode, currentSlot: currSlot })}
                                on:keydown={(e) => e.key === 'Enter' && dispatch('assignSlot', { slot: slotNum, targetNode, currentSlot: currSlot })}
                            >
                                <span class="slot-badge">{getShortcutLabel(idx)}</span>
                                <span class="slot-text">槽位 {slotNum}</span>
                                <span class="slot-status" class:is-empty={!currSlot || currSlot.type === 'empty'}>
                                    {getSlotSummary(currSlot)}
                                </span>
                            </div>
                        {/each}
                    </div>
                </div>
                <div class="menu-divider"></div>
                <div class="menu-item" on:click={() => dispatch('editText')} on:keydown={(e) => e.key === 'Enter' && dispatch('editText')}>Edit</div>
                <div class="menu-divider"></div>
            {/if}
            <div class="menu-item delete" on:click={() => dispatch('delete')} on:keydown={(e) => e.key === 'Enter' && dispatch('delete')}>Delete</div>
        {/if}
    </div>
{/if}

<style>
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
        min-width: 136px;
        z-index: 9999;
    }

    .menu-item {
        padding: 5px 10px;
        font-size: var(--app-font-size, 13px);
        border-radius: 4px;
        cursor: pointer;
        color: #333;
        text-align: left;
        display: flex;
        align-items: center;
        gap: 6px;
        transition: all 0.18s cubic-bezier(0.34, 1.56, 0.64, 1);
        user-select: none;
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

    .item-icon-svg {
        flex-shrink: 0;
        opacity: 0.85;
    }

    .menu-label {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        flex: 1;
    }

    .chevron-right {
        opacity: 0.6;
        margin-left: 8px;
    }

    /* 二级子菜单 */
    .submenu-trigger {
        position: relative;
    }

    .submenu {
        display: none;
        position: absolute;
        left: calc(100% + 4px);
        top: -4px;
        background: rgba(248, 250, 252, 0.92);
        -webkit-backdrop-filter: blur(22px) saturate(1.3);
        backdrop-filter: blur(22px) saturate(1.3);
        border: 1px solid rgba(255, 255, 255, 0.74);
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.68),
            0 10px 26px rgba(25, 41, 55, 0.18);
        border-radius: 7px;
        padding: 4px;
        min-width: 172px;
        z-index: 10000;
        flex-direction: column;
        gap: 2px;
    }

    .submenu.flip-left {
        left: auto;
        right: calc(100% + 4px);
    }

    .submenu.flip-up {
        top: auto;
        bottom: -4px;
    }

    .submenu-trigger:hover .submenu {
        display: flex;
    }

    .submenu-item {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        padding: 4px 8px;
    }

    .slot-badge {
        font-size: 10.5px;
        font-weight: 600;
        color: #4b637a;
        background: rgba(140, 170, 200, 0.16);
        border: 1px solid rgba(140, 170, 200, 0.32);
        border-radius: 3px;
        padding: 1px 4px;
        line-height: 1.2;
        flex-shrink: 0;
    }

    .slot-text {
        font-weight: 550;
        color: #1e293b;
        flex-shrink: 0;
    }

    .slot-status {
        margin-left: auto;
        color: #55697a;
        font-size: 11.5px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 70px;
    }

    .slot-status.is-empty {
        color: #94a3b8;
        font-style: italic;
    }
    
    .menu-divider {
        height: 1px;
        background: rgba(0,0,0,0.08);
        margin: 4px 0;
    }
</style>
