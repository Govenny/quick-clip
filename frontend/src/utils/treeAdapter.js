/**
 * 树结构数据适配器与操作工具库
 */

export function generateId() {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
        return crypto.randomUUID();
    }
    return 'node-' + Math.random().toString(36).slice(2, 11) + '-' + Date.now().toString(36);
}

/**
 * 智能规范化树数据
 * 支持：
 * 1. 标准 TreeNode 格式: { id, name, type, value, children }
 * 2. 旧版格式: [ { "test2": "123" }, { "先吃饭": [ ... ] } ]
 * 3. 过滤并清除历史遗留的空对象 {} 脏数据
 *
 * @param {any} rawData 原始数据
 * @returns {Array<object>} 规范化后的树节点数组
 */
export function normalizeTree(rawData) {
    if (!Array.isArray(rawData)) {
        return [];
    }

    const result = [];

    for (const item of rawData) {
        if (!item || typeof item !== 'object') {
            continue;
        }

        // 已经符合标准 TreeNode 格式
        if (typeof item.name === 'string' && (item.type === 'folder' || item.type === 'text')) {
            const node = {
                id: item.id || generateId(),
                name: item.name,
                type: item.type,
            };

            if (item.type === 'folder') {
                node.children = normalizeTree(item.children || []);
            } else {
                node.value = typeof item.value === 'string' ? item.value : (item.value != null ? String(item.value) : '');
            }
            result.push(node);
            continue;
        }

        // 旧版单键值对格式: { [key]: value }
        const entries = Object.entries(item);
        if (entries.length === 0) {
            // 过滤空对象 {}
            continue;
        }

        for (const [key, val] of entries) {
            // 文件夹：值是数组
            if (Array.isArray(val)) {
                result.push({
                    id: generateId(),
                    name: key,
                    type: 'folder',
                    children: normalizeTree(val)
                });
            } else {
                // 文本：值是字符串或其它普通值
                result.push({
                    id: generateId(),
                    name: key,
                    type: 'text',
                    value: typeof val === 'string' ? val : (val != null ? String(val) : '')
                });
            }
        }
    }

    return result;
}

/**
 * 将规范树转换为旧版持久化格式（用于向后兼容导出或需要旧格式的场景）
 * @param {Array<object>} nodes
 * @returns {Array<object>}
 */
export function toLegacyTree(nodes) {
    if (!Array.isArray(nodes)) return [];
    const result = [];

    for (const node of nodes) {
        if (!node) continue;
        if (node.type === 'folder') {
            result.push({
                [node.name]: toLegacyTree(node.children || [])
            });
        } else {
            result.push({
                [node.name]: node.value || ''
            });
        }
    }

    return result;
}

/**
 * 根据 id 查找节点
 * @param {Array<object>} nodes
 * @param {string} id
 * @returns {object | null}
 */
export function findNodeById(nodes, id) {
    if (!Array.isArray(nodes) || !id) return null;

    for (const node of nodes) {
        if (node.id === id) return node;
        if (node.type === 'folder' && Array.isArray(node.children)) {
            const found = findNodeById(node.children, id);
            if (found) return found;
        }
    }
    return null;
}

/**
 * 查找节点的父级及索引
 * @param {Array<object>} nodes
 * @param {string} id
 * @param {object | null} parentNode
 * @returns {{ parent: object | null, array: Array<object>, index: number } | null}
 */
export function findParent(nodes, id, parentNode = null) {
    if (!Array.isArray(nodes) || !id) return null;

    for (let i = 0; i < nodes.length; i++) {
        if (nodes[i].id === id) {
            return {
                parent: parentNode,
                array: nodes,
                index: i
            };
        }
        if (nodes[i].type === 'folder' && Array.isArray(nodes[i].children)) {
            const found = findParent(nodes[i].children, id, nodes[i]);
            if (found) return found;
        }
    }

    return null;
}

/**
 * 检查 targetId 是否是 sourceId 的后代节点（防止父节点拖入子节点形成死循环）
 * @param {object} sourceNode
 * @param {string} targetId
 * @returns {boolean}
 */
export function isDescendant(sourceNode, targetId) {
    if (!sourceNode || sourceNode.type !== 'folder' || !Array.isArray(sourceNode.children)) {
        return false;
    }

    for (const child of sourceNode.children) {
        if (child.id === targetId) return true;
        if (isDescendant(child, targetId)) return true;
    }
    return false;
}

/**
 * 根据 ID 安全删除节点（直接从父数组中 splice，彻底避免空对象残留）
 * @param {Array<object>} nodes
 * @param {string} id
 * @returns {boolean} 是否删除成功
 */
export function deleteNodeById(nodes, id) {
    const info = findParent(nodes, id);
    if (!info) return false;

    info.array.splice(info.index, 1);
    return true;
}

/**
 * 移动节点（支持拖拽排序与放入文件夹）
 * @param {Array<object>} nodes 根树节点数组
 * @param {string} sourceId 被拖拽节点的 ID
 * @param {string} targetId 目标节点的 ID
 * @param {'before' | 'after' | 'inside'} position 放置位置
 * @returns {boolean} 是否移动成功
 */
export function moveNode(nodes, sourceId, targetId, position) {
    if (!sourceId || !targetId || sourceId === targetId) return false;

    const sourceInfo = findParent(nodes, sourceId);
    if (!sourceInfo) return false;

    const sourceNode = sourceInfo.array[sourceInfo.index];

    // 防止将父节点移动到自身子树中
    if (sourceNode.type === 'folder' && isDescendant(sourceNode, targetId)) {
        return false;
    }

    const targetInfo = findParent(nodes, targetId);
    if (!targetInfo) return false;

    const targetNode = targetInfo.array[targetInfo.index];

    // 1. 从原位置移出
    sourceInfo.array.splice(sourceInfo.index, 1);

    // 2. 插入到新位置
    if (position === 'inside') {
        if (targetNode.type !== 'folder') return false;
        if (!Array.isArray(targetNode.children)) {
            targetNode.children = [];
        }
        targetNode.children.push(sourceNode);
        return true;
    }

    // 'before' 或 'after'：插入到 targetNode 所在的同级数组中
    const destArray = targetInfo.array;
    // 重新获取 targetIndex
    let destIndex = destArray.findIndex(n => n.id === targetId);
    if (destIndex === -1) return false;

    if (position === 'after') {
        destIndex += 1;
    }

    destArray.splice(destIndex, 0, sourceNode);
    return true;
}

/**
 * 递归搜索节点
 * @param {Array<object>} nodes
 * @param {string} query
 * @param {string} path
 * @returns {Array<{ name: string, content: string, fullPath: string }>}
 */
export function searchTree(nodes, query, path = "") {
    if (!Array.isArray(nodes) || !query || !query.trim()) return [];

    let results = [];
    const q = query.trim().toLowerCase();

    for (const node of nodes) {
        if (!node) continue;
        if (node.type === 'folder') {
            const subResults = searchTree(node.children || [], query, path + node.name + " > ");
            results = [...results, ...subResults];
        } else if (node.type === 'text') {
            if (node.name.toLowerCase().includes(q)) {
                results.push({
                    id: node.id,
                    name: node.name,
                    content: node.value || '',
                    fullPath: path + node.name
                });
            }
        }
    }

    return results;
}

