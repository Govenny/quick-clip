<script>
    import { createEventDispatcher, tick } from 'svelte';
    import { fade, fly } from 'svelte/transition';
    import { quartOut, cubicOut } from 'svelte/easing';

    export let visible = false;
    export let isEdit = false;
    export let initialTitle = "";
    export let initialValue = "";

    const dispatch = createEventDispatcher();

    let titleName = "";
    let textName = "";
    let titleInputRef;
    let textInputRef;

    function initModal(node) {
        titleName = initialTitle || "";
        textName = initialValue || "";
        setTimeout(() => {
            titleInputRef?.focus();
            if (isEdit) {
                titleInputRef?.select();
            }
        }, 50);
    }

    $: isFormValid = titleName.trim() !== "" && textName.trim() !== "";

    function handleKeyDown(event, isTitleInput) {
        const { key } = event;

        if (key === 'Escape') {
            event.preventDefault();
            dispatch('cancel');
            return;
        }

        if (key === 'Enter') {
            if (isTitleInput) {
                event.preventDefault();
                textInputRef?.focus();
                return;
            }

            // 内容框使用 Shift+Enter 换行；Enter 保存
            if (event.shiftKey) {
                return;
            }

            event.preventDefault();
            if (isFormValid) {
                dispatch('submit', {
                    title: titleName.trim(),
                    value: textName
                });
            } else {
                alert("请完善输入");
            }
        } else if (key === 'Tab' && isTitleInput && !event.shiftKey) {
            event.preventDefault();
            textInputRef?.focus();
        }
    }
</script>

{#if visible}
    <div 
        class="modal-overlay" 
        on:keydown={(e) => e.key === 'Escape' && dispatch('cancel')} 
        on:click={() => dispatch('cancel')} 
        in:fade={{ duration: 130, easing: quartOut }} 
        out:fade={{ duration: 80 }}
    >
        <div 
            class="modal-box" 
            use:initModal
            on:keydown|stopPropagation 
            on:click|stopPropagation 
            in:fly={{ y: 15, duration: 230, easing: cubicOut }} 
            out:fly={{ y: 10, duration: 100 }}
        >
            <div class="input-group">
                <input 
                    type="text" 
                    class="title-input" 
                    bind:value={titleName} 
                    bind:this={titleInputRef} 
                    placeholder={isEdit ? "Edit Key / Name" : "Key / Name"} 
                    on:keydown={(e) => handleKeyDown(e, true)}
                />
                <textarea 
                    class="value-input" 
                    bind:value={textName} 
                    bind:this={textInputRef} 
                    placeholder={isEdit ? "Edit Value / Command" : "Value / Command"} 
                    spellcheck="false" 
                    on:keydown={(e) => handleKeyDown(e, false)}
                ></textarea>
            </div>
            <div class="modal-footer">
                <span class="hint">{isEdit ? "Shift+Enter for newline / Enter to update" : "Shift+Enter for newline / Enter to save"}</span>
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
        width: 380px;
        border-radius: 8px;
        box-shadow:
            inset 0 1px 0 rgba(255, 255, 255, 0.72),
            0 14px 38px rgba(24, 39, 52, 0.2);
        border: 1px solid rgba(255, 255, 255, 0.72);
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .input-group {
        display: flex;
        flex-direction: column;
    }

    .input-group input,
    .input-group textarea {
        box-sizing: border-box;
        border: none;
        padding: 12px 16px;
        font-size: 14px;
        outline: none;
        width: 100%;
        background: transparent;
        color: #293b4a;
        transition: background-color 0.18s ease;
    }

    .input-group textarea {
        min-height: 112px;
        max-height: 220px;
        resize: vertical;
        line-height: 1.45;
        font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
        overflow-y: auto;
        overflow-x: hidden;
    }

    .title-input {
        border-bottom: 1px solid #eee !important;
        font-weight: 500;
    }

    .modal-footer {
        background: #f9fafb;
        padding: 6px 10px;
        text-align: right;
        border-top: 1px solid #f0f0f0;
    }

    .hint {
        font-size: 11px;
        color: #999;
    }
</style>
