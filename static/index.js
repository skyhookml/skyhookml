import utils from './utils.js';

import Datasets from './datasets.js';
import Dataset from './dataset.js';

import Annotate from './annotate.js';
import AnnotateInt from './annotate-int.js';
import AnnotateShape from './annotate-shape.js';

import Models from './models.js';
import MArch from './m-architecture.js';
import MComp from './m-component.js';
import TrainPytorch from './m-train-pytorch.js';
import TrainYolov3 from './m-train-yolov3.js';

import Queries from './queries.js';
import ExecModel from './exec-edit-model.js';
import ExecPython from './exec-edit-python.js';
import ExecVideoSample from './exec-edit-video_sample.js';
import Compare from './compare.js';

const router = new VueRouter({
	routes: [
		{path: '/ws/:ws', component: Datasets},

		{path: '/ws/:ws/datasets', component: Datasets},
		{path: '/ws/:ws/datasets/:dsid', component: Dataset},

		{path: '/ws/:ws/annotate', component: Annotate},
		{path: '/ws/:ws/annotate/int/:setid', component: AnnotateInt},
		{path: '/ws/:ws/annotate/shape/:setid', component: AnnotateShape},

		{path: '/ws/:ws/models', component: Models},
		{path: '/ws/:ws/models/arch/:archid', component: MArch},
		{path: '/ws/:ws/models/comp/:compid', component: MComp},
		{path: '/ws/:ws/train/pytorch/:nodeid', component: TrainPytorch},
		{path: '/ws/:ws/train/yolov3/:nodeid', component: TrainYolov3},

		{path: '/ws/:ws/queries', component: Queries},
		{path: '/ws/:ws/exec/model/:nodeid', component: ExecModel},
		{path: '/ws/:ws/exec/python/:nodeid', component: ExecPython},
		{path: '/ws/:ws/exec/video_sample/:nodeid', component: ExecVideoSample},
		{path: '/ws/:ws/compare/:nodeid/:otherws/:othernodeid', component: Compare},
	],
});

const globals = {};
Vue.prototype.$globals = globals;
Promise.all([
	utils.request(null, 'GET', '/data-types', null, (dataTypes) => {
		globals.dataTypes = dataTypes;
	}),
]).then(() => {
	const app = new Vue({
		el: '#app',
		data: {
			error: '',
			selectedWorkspace: '',
			workspaces: [],
			addForms: null,
		},
		created: function() {
			this.fetch();
			this.resetForm();

			if(this.$route.params.ws) {
				this.selectedWorkspace = this.$route.params.ws;
			}
		},
		methods: {
			fetch: function() {
				utils.request(this, 'GET', '/workspaces', null, (data) => {
					this.workspaces = data;
				});
			},
			resetForm: function() {
				this.addForms = {
					workspace: {
						name: '',
					},
				};
			},
			setPage: function(name) {
				if(!this.$route.params.ws) {
					return;
				}
				this.$router.push('/ws/'+this.$route.params.ws+'/'+name);
				this.setError('');
			},
			changedWorkspace: function() {
				this.$router.push('/ws/'+this.selectedWorkspace);
				this.resetForm();
			},
			createWorkspace: function() {
				let name = this.addForms.workspace.name;
				utils.request(this, 'POST', '/workspaces', {name: name}, () => {
					this.resetForm();
					this.fetch();
					this.$router.push('/ws/'+name);
				});
			},
			cloneWorkspace: function() {
				let url = '/workspaces/'+this.$route.params.ws+'/clone';
				var params = {
					name: this.addForms.workspace.name,
				};
				utils.request(this, 'POST', url, params, () => {
					this.resetForm();
					this.fetch();
					this.$router.push('/ws/'+params.name);
				});
			},
			deleteWorkspace: function() {
				utils.request(this, 'DELETE', '/workspaces/'+this.selectedWorkspace, null, () => {
					this.fetch();
					this.$router.push('/');
				});
			},
			setError: function(error) {
				this.error = error;
			},
		},
		router: router,
	});

	$(document).ready(function() {
		$('body').keypress(function(e) {
			app.$emit('keypress', e);
		});
		$('body').keyup(function(e) {
			app.$emit('keyup', e);
		});
	});
});
