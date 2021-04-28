import utils from './utils.js';
import App from './app.vue';

const globals = {};
Vue.prototype.$globals = globals;
Promise.all([
	utils.request(null, 'GET', '/data-types', null, (dataTypes) => {
		globals.dataTypes = dataTypes;
	}),
	utils.request(null, 'GET', '/ops', null, (ops) => {
		globals.ops = ops;
	}),
]).then(() => {
	const app = new Vue(App);
	app.$mount('#app');
	globals.app = app;

	$(document).ready(function() {
		$('body').keypress(function(e) {
			app.$emit('keypress', e);
		});
		$('body').keyup(function(e) {
			app.$emit('keyup', e);
		});
	});
});

// enable bootstrap tooltips
new bootstrap.Tooltip(document.body, {
	selector: '[data-bs-toggle="tooltip"]',
});
