<script>
    import { createEventDispatcher } from 'svelte';
    import { scale, fade } from 'svelte/transition';

    export let visible = false;
    export let x = 0;
    export let y = 0;
    export let flipX = false;
    export let flipY = false;
    export let targetNode = null;

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
</script>

{#if visible}
    <div 
        class="context-menu"
        style="position: fixed; top: {y}px; left: {x}px; transform-origin: {flipX ? 'right' : 'left'} {flipY ? 'bottom' : 'top'};"
        in:scale={{ duration: 130, easing: iosElastic }} 
        out:fade={{ duration: 60 }}
        on:contextmenu|preventDefault
    >
        {#if targetNode?.type === 'folder'}
            <div class="menu-item" on:click={() => dispatch('addText')} on:keydown={(e) => e.key === 'Enter' && dispatch('addText')}>New Text</div>
            <div class="menu-item" on:click={() => dispatch('addDir')} on:keydown={(e) => e.key === 'Enter' && dispatch('addDir')}>New Folder</div>
            <div class="menu-divider"></div>
            <div class="menu-item" on:click={() => dispatch('editDir')} on:keydown={(e) => e.key === 'Enter' && dispatch('editDir')}>Edit</div>
            <div class="menu-divider"></div>
        {/if}
        {#if targetNode?.type === 'text'}
            <div class="menu-item" on:click={() => dispatch('editText')} on:keydown={(e) => e.key === 'Enter' && dispatch('editText')}>Edit</div>
            <div class="menu-divider"></div>
        {/if}
        <div class="menu-item delete" on:click={() => dispatch('delete')} on:keydown={(e) => e.key === 'Enter' && dispatch('delete')}>Delete</div>
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
        min-width: 128px;
        z-index: 9999;
    }

    .menu-item {
        padding: 4px 10px;
        font-size: var(--app-font-size, 13px);
        border-radius: 4px;
        cursor: pointer;
        color: #333;
        text-align: left;
        transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
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
    
    .menu-divider {
        height: 1px;
        background: rgba(0,0,0,0.1);
        margin: 4px 0;
    }
</style>

