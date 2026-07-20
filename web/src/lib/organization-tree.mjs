export function replaceTreeNodeChildren(nodes, key, children) {
  let changed = false;
  const nextNodes = nodes.map((node) => {
    if (node.key === key) {
      changed = true;
      return { ...node, children };
    }
    if (!node.children?.length) return node;

    const nextChildren = replaceTreeNodeChildren(node.children, key, children);
    if (nextChildren === node.children) return node;

    changed = true;
    return { ...node, children: nextChildren };
  });

  return changed ? nextNodes : nodes;
}
