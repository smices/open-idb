import assert from 'node:assert/strict';
import test from 'node:test';

import { replaceTreeNodeChildren } from './organization-tree.mjs';

test('replaceTreeNodeChildren immutably nests loaded children under the matching node', () => {
  const root = {
    key: 'company:root',
    title: 'SweetNight · company',
    raw: { kind: 'company', id: 'root' },
  };
  const sibling = {
    key: 'department:other',
    title: 'Other · department',
    raw: { kind: 'department', id: 'other' },
  };
  const children = [{
    key: 'department:ccbg',
    title: 'CCBG · department',
    raw: { kind: 'department', id: 'ccbg' },
  }];

  const initial = [root, sibling];
  const result = replaceTreeNodeChildren(initial, root.key, children);

  assert.notStrictEqual(result, initial);
  assert.notStrictEqual(result[0], root);
  assert.deepEqual(result[0].children, children);
  assert.equal(result[1], sibling);
});

test('replaceTreeNodeChildren preserves the original tree when the key is absent', () => {
  const initial = [{
    key: 'company:root',
    children: [{ key: 'department:ccbg' }],
  }];

  assert.equal(
    replaceTreeNodeChildren(initial, 'department:missing', []),
    initial,
  );
});

test('replaceTreeNodeChildren updates a nested department without changing its siblings', () => {
  const department = { key: 'department:ccbg' };
  const root = { key: 'company:root', children: [department] };
  const initial = [root];
  const children = [{ key: 'department:ctb' }];

  const result = replaceTreeNodeChildren(initial, department.key, children);

  assert.notStrictEqual(result, initial);
  assert.notStrictEqual(result[0], root);
  assert.notStrictEqual(result[0].children[0], department);
  assert.deepEqual(result[0].children[0].children, children);
});
