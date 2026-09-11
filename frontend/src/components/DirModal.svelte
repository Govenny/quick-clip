<script>
    import { createEventDispatcher, tick } from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { quartOut, cubicOut } from 'svelte/easing';

    export let visible = false;
    export let isEdit = false;
    export let initialName = "";

    const dispatch = createEventDispatcher();

    let dirName = "";
    let dirInputRef;

    function initModal(node) {
        dirName = initialName || "";
        setTimeout(() => {
            dirInputRef?.focus();
            if (isEdit) {
                dirInputRef?.select();
            }
        }, 50);
    }

    function handleConfirm() {
        const trimmed = dirName.trim();
        if (!trimmed) {
            alert("名称无效");
            return;
        }
        dispatch('submit', { name: trimmed });
    }

    function handleKeyDown(event) {
        if (event.key === 'Enter') {
            event.preventDefault();
            handleConfirm();
        } else if (event.key === 'Escape') {
            event.preventDefault();
            dispatch('cancel');
        }
    }
</script>

{#if visible}
    <div 
        class="modal-overlay" 
        on:click={() => dispatch('cancel')} 
        on:keydown={(e) => e.key === 'Escape' && dispatch('cancel')} 
        in:fade={{ duration: 130, easing: quartOut }} 
        out:fade={{ duration: 80 }}
    >
        <div 
            class="modal-box compact" 
            use:initModal
            on:click|stopPropagation 
            on:keydown|stopPropagation 
            in:fly={{ y: 15, duration: 230, easing: cubicOut }} 
            out:fly={{ y: 10, duration: 100 }}
        >
            <input 
                type="text" 
                bind:value={dirName} 
                bind:this={dirInputRef} 
                placeholder={isEdit ? "Rename Folder" : "Folder Name"} 
                on:keydown={handleKeyDown}
            />
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0; left: 0; right: 0; bottom: 0;
        background: rgba(215, 225, 233, 0.28);
        -webkit-backdrop-filter: blur(8px) saturate(1.12);
        backdrop-filter: blur(8px) saturate(1.12);
        display: flex;
        align-items: flex-start;
        justify-content: center;
        padding-top: 80px;
        z-index: 9998;
    }

    .modal-box {
        background: rgba(249, 251, 252, 0.86);
        -webkit-backdrop-filter: blur(26px) saturate(1.25);
        backdrop-filter: blur(26px) saturate(1.25);
        border-radius: 8px;
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.72),
            0 14px 38px rgba(24, 39, 52, 0.2);
        border: 1px solid rgba(255, 255, 255, 0.72);
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .modal-box.compact {
        width: 300px;
        padding: 8px;
    }

    .modal-box.compact input {
        border: 1px solid #eee;
        border-radius: 4px;
        padding: 6px 10px;
        background: #f9f9f9;
        font-size: 13px;
        width: 100%;
        box-sizing: border-box;
        outline: none;
        color: #293b4a;
        transition: all 0.18s ease;
    }

    .modal-box.compact input:focus {
        background: #fff;
        border-color: #3b82f6;
    }
</style>
