import globals from "globals";
import eslint from '@eslint/js';
import unicorn from 'eslint-plugin-unicorn';
import sonarjs from 'eslint-plugin-sonarjs';
import importX from 'eslint-plugin-import-x';

export default [
	{
		ignores: [
			'eslint.config.js',
			'node_modules/',
			'dist/',
			'build/',
			'coverage/',
		],
	},

	{
		languageOptions: {
			globals: globals.node,
		},
	},

	eslint.configs.recommended,

	unicorn.configs.recommended,

	sonarjs.configs.recommended,

	{
		plugins: {
			'import-x': importX,
		},

		rules: {
			// Keep individual functions small.
			'max-lines-per-function': [
				'error',
				{
					max: 50,
					skipBlankLines: true,
					skipComments: true,
				},
			],

			'max-lines': [
				'error',
				{
					max: 1000,
					skipBlankLines: true,
					skipComments: true,
				},
			],

			// Cyclomatic complexity.
			complexity: ['error', 10],

			// Don't build the pyramid of doom.
			'max-depth': ['error', 3],

			// Functions requiring too many things are usually doing too much.
			'max-params': ['error', 4],

			// Limit procedural blobs.
			'max-statements': ['error', 20],

			// Don't bury callbacks inside callbacks.
			'max-nested-callbacks': ['error', 3],

			// Control flow
			'no-else-return': 'error',
			'no-lonely-if': 'error',
			'no-nested-ternary': 'error',
			'no-unneeded-ternary': 'error',
			'no-unreachable-loop': 'error',
			'no-fallthrough': 'error',
			'no-constant-condition': 'error',
			'no-duplicate-case': 'error',
			'no-duplicate-imports': 'error',
			'no-self-compare': 'error',
			'no-unmodified-loop-condition': 'error',

			'default-case-last': 'error',
			'default-param-last': 'error',
			'consistent-return': 'error',

			// JavaScript correctness
			'no-await-in-loop': 'error',
			'no-promise-executor-return': 'error',
			'no-return-await': 'error',
			'no-self-assign': 'error',
			'no-throw-literal': 'error',
			'no-useless-catch': 'error',
			'require-atomic-updates': 'error',

			// Variables / mutability
			'prefer-const': 'error',
			'no-var': 'error',
			'no-delete-var': 'error',
			'no-shadow': 'error',
			'no-use-before-define': [
				'error',
				{
					functions: false,
					classes: true,
					variables: true,
				},
			],

			// Clean code
			'no-multi-assign': 'error',
			'no-param-reassign': 'error',
			'no-return-assign': 'error',
			'no-sequences': 'error',
			'no-unneeded-ternary': 'error',
			'no-useless-call': 'error',
			'no-useless-concat': 'error',
			'no-useless-return': 'error',

			'object-shorthand': 'error',
			'prefer-template': 'error',
			'prefer-object-spread': 'error',
			'prefer-destructuring': [
				'error',
				{
					object: true,
					array: false,
				},
			],

			// Functions
			'func-style': ['error', 'declaration', { allowArrowFunctions: true }],

			// Naming / readability
			'id-length': [
				'error',
				{
					min: 2,
					exceptions: [
						'i',
						'j',
						'k',
						'x',
						'y',
						'z',
						'_',
					],
				},
			],

			// Imports / architecture
			'import-x/no-duplicates': 'error',
			'import-x/no-named-as-default': 'error',
			'import-x/no-named-as-default-member': 'error',
			'import-x/no-unresolved': 'error',

			// Formatting-ish rules
			'comma-dangle': ['error', 'always-multiline'],
			'curly': ['error', 'all'],
			'eqeqeq': ['error', 'always'],
			'no-mixed-operators': 'error',
			'no-multiple-empty-lines': [
				'error',
				{
					max: 1,
					maxEOF: 0,
				},
			],
			'no-trailing-spaces': 'error',
			'semi': ['error', 'always'],
			'quotes': [
				'error',
				'single',
				{
					avoidEscape: true,
					allowTemplateLiterals: false,
				},
			],

			// Comments / TODOs
			'no-warning-comments': [
				'error',
				{
					terms: ['todo', 'fixme', 'hack', 'xxx'],
					location: 'anywhere',
				},
			],
		},
	},
];
