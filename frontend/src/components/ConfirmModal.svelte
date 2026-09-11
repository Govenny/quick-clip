<script>
    import { createEventDispatcher } from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { quartOut, cubicOut } from 'svelte/easing';

    export let visible = false;
    export let title = "Confirm Delete";
    export let message = "";
    export let confirmText = "Delete";
    export let cancelText = "Cancel";

    const dispatch = createEventDispatcher();
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
            class="modal-box compact confirm-modal" 
            on:click|stopPropagation 
            on:keydown|stopPropagation 
            in:fly={{ y: 15, duration: 230, easing: cubicOut }} 
            out:fly={{ y: 10, duration: 100 }}
        >
            <div class="confirm-content">
                <div class="confirm-text">
                    <div class="confirm-title">{title}</div>
                    {#if message}
                        <div class="confirm-message">{message}</div>
                    {/if}
                </div>
            </div>
            <div class="modal-footer confirm-footer">
                <button class="btn btn-cancel" on:click={() => dispatch('cancel')}>{cancelText}</button>
                <button class="btn btn-delete" on:click={() => dispatch('confirm')}>{confirmText}</button>
            </div>
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

    .confirm-modal {
        width: 320px;
        padding: 16px;
    }

    .confirm-content {
        display: flex;
        align-items: flex-start;
        gap: 15px;
        margin-bottom: 20px;
    }

    .confirm-text {
        flex: 1;
    }

    .confirm-title {
        font-size: 15px;
        font-weight: 600;
        color: #293b4a;
        margin-bottom: 8px;
    }

    .confirm-message {
        font-size: 13px;
        color: #5c7080;
        line-height: 1.5;
        word-break: break-all;
    }

    .confirm-footer {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        padding: 0;
        background: transparent;
        border-top: none;
    }

    .btn {
        padding: 6px 14px;
        border-radius: 5px;
        font-size: 13px;
        cursor: pointer;
        border: 1px solid transparent;
        transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
        font-family: inherit;
    }

    .btn-cancel {
        background: rgba(255, 255, 255, 0.6);
        color: #55697a;
        border-color: rgba(75, 99, 119, 0.2);
    }

    .btn-cancel:hover {
        background: rgba(255, 255, 255, 0.9);
        color: #223240;
        border-color: rgba(75, 99, 119, 0.35);
    }

    .btn-delete {
        background: rgba(239, 68, 68, 0.12);
        color: #dc2626;
        border-color: rgba(239, 68, 68, 0.3);
    }

    .btn-delete:hover {
        background: #ef4444;
        color: #fff;
        border-color: #dc2626;
    }
</style>
