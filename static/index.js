import utils from './utils.js';

import Datasets from './datasets.js';
import Dataset from './dataset.js';

import Annotate from './annotate.js';
import AnnotateInt from './annotate-int.js';
import AnnotateShape from './annotate-shape.js';
import AnnotateDetectionToTrack from './annotate-detection-to-track.js';

import Models from './models.js';
import MArch from './m-architecture.js';
import MComp from './m-component.js';

import Queries from './queries.js';
import ExecCropResize from './exec-edit-cropresize.js';
import ExecDetectionFilter from './exec-edit-detection_filter.js';
import ExecResample from './exec-edit-resample.js';
import ExecSimpleTracker from './exec-edit-simple_tracker.js';
import ExecReidTracker from './exec-edit-reid_tracker.js';
import ExecPython from './exec-edit-python.js';
import ExecPytorchTrain from './exec-edit-pytorch_train.js';
import ExecPytorchInfer from './exec-edit-pytorch_infer.js';
import ExecYolov3Train from './exec-edit-yolov3_train.js';
import ExecYolov3Infer from './exec-edit-yolov3_infer.js';
import ExecUnsupervisedReid from './exec-edit-pytorch_train.js';
import ExecVideoSample from './exec-edit-video_sample.js';
import Compare from './compare.js';
import Interactive from './interactive.js';

import Jobs from './jobs.js';
import JobExecNode from './job-exec_node.js';
import JobPytorchTrain from './job-pytorch_train.js';

const router = new VueRouter({
	routes: [
		{path: '/ws/:ws', component: Datasets},

		{path: '/ws/:ws/datasets', component: Datasets},
		{path: '/ws/:ws/datasets/:dsid', component: Dataset},

		{path: '/ws/:ws/annotate', component: Annotate},
		{path: '/ws/:ws/annotate/int/:setid', component: AnnotateInt},
		{path: '/ws/:ws/annotate/shape/:setid', component: AnnotateShape},
		{path: '/ws/:ws/annotate/detection-to-track/:setid', component: AnnotateDetectionToTrack},

		{path: '/ws/:ws/models', component: Models},
		{path: '/ws/:ws/models/arch/:archid', component: MArch},
		{path: '/ws/:ws/models/comp/:compid', component: MComp},

		{path: '/ws/:ws/queries', component: Queries},
		{path: '/ws/:ws/exec/cropresize/:nodeid', component: ExecCropResize},
		{path: '/ws/:ws/exec/detection_filter/:nodeid', component: ExecDetectionFilter},
		{path: '/ws/:ws/exec/simple_tracker/:nodeid', component: ExecSimpleTracker},
		{path: '/ws/:ws/exec/reid_tracker/:nodeid', component: ExecReidTracker},
		{path: '/ws/:ws/exec/resample/:nodeid', component: ExecResample},
		{path: '/ws/:ws/exec/python/:nodeid', component: ExecPython},
		{path: '/ws/:ws/exec/pytorch_train/:nodeid', component: ExecPytorchTrain},
		{path: '/ws/:ws/exec/pytorch_infer/:nodeid', component: ExecPytorchInfer},
		{path: '/ws/:ws/exec/yolov3_train/:nodeid', component: ExecYolov3Train},
		{path: '/ws/:ws/exec/yolov3_infer/:nodeid', component: ExecYolov3Infer},
		{path: '/ws/:ws/exec/unsupervised_reid/:nodeid', component: ExecUnsupervisedReid},
		{path: '/ws/:ws/exec/video_sample/:nodeid', component: ExecVideoSample},
		{path: '/ws/:ws/compare/:nodeid/:otherws/:othernodeid', component: Compare},
		{path: '/ws/:ws/interactive/:nodeid', component: Interactive},

		{path: '/ws/:ws/jobs', component: Jobs},
		{path: '/ws/:ws/jobs/execnode/:jobid', component: JobExecNode},
		{path: '/ws/:ws/jobs/pytorch_train/:jobid', component: JobPytorchTrain},
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
