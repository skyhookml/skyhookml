import utils from './utils.js';

import Dashboard from './dashboard.js';

import QuickstartImport from './quickstart-import.js';
import QuickstartTrain from './quickstart-train.js';
import QuickstartApply from './quickstart-apply.js';

import Datasets from './datasets.js';
import Dataset from './dataset.js';
import RenderItem from './render-item.js';

import Annotate from './annotate.js';
import AnnotateAdd from './annotate-add.js';
import AnnotateInt from './annotate-int.js';
import AnnotateShape from './annotate-shape.js';
import AnnotateDetectionToTrack from './annotate-detection-to-track.js';

import Models from './models.js';
import MArch from './m-architecture.js';
import MComp from './m-component.js';

import Queries from './queries.js';
import ExecEdit from './exec-edit.js';
import Compare from './compare.js';
import Interactive from './interactive.js';

import Jobs from './jobs.js';
import Job from './job.js';

const router = new VueRouter({
	routes: [
		{path: '/', redirect: '/ws/default'},

		{path: '/ws/:ws', component: Dashboard},

		{path: '/ws/:ws/quickstart/import', component: QuickstartImport},
		{path: '/ws/:ws/quickstart/train', component: QuickstartTrain},
		{path: '/ws/:ws/quickstart/apply', component: QuickstartApply},

		{path: '/ws/:ws/datasets', component: Datasets},
		{path: '/ws/:ws/datasets/:dsid', component: Dataset},
		{path: '/ws/:ws/datasets/:dsid/items/:itemkey', component: RenderItem},

		{path: '/ws/:ws/annotate', component: Annotate},
		{path: '/ws/:ws/annotate-add', component: AnnotateAdd},
		{path: '/ws/:ws/annotate/int/:setid', component: AnnotateInt},
		{path: '/ws/:ws/annotate/shape/:setid', component: AnnotateShape},
		{path: '/ws/:ws/annotate/detection-to-track/:setid', component: AnnotateDetectionToTrack},

		{path: '/ws/:ws/models', component: Models},
		{path: '/ws/:ws/models/arch/:archid', component: MArch},
		{path: '/ws/:ws/models/comp/:compid', component: MComp},

		{path: '/ws/:ws/queries', component: Queries},
		{path: '/ws/:ws/exec/:nodeid', component: ExecEdit},
		{path: '/ws/:ws/compare/:nodeid/:otherws/:othernodeid', component: Compare},
		{path: '/ws/:ws/interactive/:nodeid', component: Interactive},

		{path: '/ws/:ws/jobs', component: Jobs},
		{path: '/ws/:ws/jobs/:jobid', component: Job},
	],
});

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
		computed: {
			wsPrefix: function() {
				if(this.$route.params.ws) {
					return '/ws/' + this.$route.params.ws;
				} else if(this.selectedWorkspace) {
					return '/ws' + this.selectedWorkspace;
				} else if(this.workspaces.length > 0) {
					return '/ws/' + this.workspaces[0];
				} else {
					return '/';
				}
			},
		},
		router: router,
	});
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
