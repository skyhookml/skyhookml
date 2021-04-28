const { VueLoaderPlugin } = require('vue-loader');
const path = require('path');

module.exports = {
	mode: 'development',
	entry: './src/index.js',
	module: {
		rules: [
			{
				test: /\.vue$/,
				loader: 'vue-loader',
			},
		],
	},
	plugins: [
		// make sure to include the plugin!
		new VueLoaderPlugin(),
	],
	output: {
		filename: 'main.js',
		path: path.resolve(__dirname, 'dist'),
	},
};
