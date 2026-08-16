module.exports = {
  root: true,
  env: {
    browser: true,
    es2022: true,
    node: true,
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-recommended',
    '@vue/eslint-config-typescript',
  ],
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module',
  },
  rules: {
    'vue/multi-word-component-names': 'off', // 页面文件允许单字（index.vue）
    'vue/max-attributes-per-line': 'off',
    // 允许单行元素（紧凑布局是 06-UI 规范的要求）
    'vue/singleline-html-element-content-newline': 'off',
    'vue/html-self-closing': ['warn', { html: { void: 'always' } }],
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
  },
  ignorePatterns: ['dist', 'node_modules', '*.config.js', 'tailwind.config.js'],
}
