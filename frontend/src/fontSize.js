export const FONT_SIZE_LEVELS = {
    1: { level: 1, name: '较小', fontSize: '12px', chipHeight: '22px', itemHeight: '26px', iconSize: '10px', window: '395 × 550' },
    2: { level: 2, name: '标准', fontSize: '13px', chipHeight: '24px', itemHeight: '28px', iconSize: '11px', window: '420 × 580' },
    3: { level: 3, name: '中等', fontSize: '14.5px', chipHeight: '26px', itemHeight: '31px', iconSize: '12px', window: '450 × 615' },
    4: { level: 4, name: '较大', fontSize: '16px', chipHeight: '28px', itemHeight: '34px', iconSize: '13px', window: '480 × 645' },
    5: { level: 5, name: '特大', fontSize: '18px', chipHeight: '30px', itemHeight: '37px', iconSize: '14px', window: '515 × 680' },
};

export function applyFontSizeLevel(level) {
    const lvlNum = Math.max(1, Math.min(5, Number(level) || 2));
    const lvl = FONT_SIZE_LEVELS[lvlNum] || FONT_SIZE_LEVELS[2];
    const root = document.documentElement;
    root.style.setProperty('--app-font-size', lvl.fontSize);
    root.style.setProperty('--app-chip-height', lvl.chipHeight);
    root.style.setProperty('--app-item-line-height', lvl.itemHeight);
    root.style.setProperty('--app-icon-size', lvl.iconSize);
}
