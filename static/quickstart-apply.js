import utils from './utils.js';
import JobConsoleProgress from './job-consoleprogress.js';
import tasks from './quickstart-model-tasks.js';
import get_parent_options from './get-parent-options.js';

export default {
	components: {
		'job-consoleprogress': JobConsoleProgress,
	},
	data: function() {
		return {
			tasks: tasks,

			// potential customParent/inputParent options
			// these are either datasets or ExecNodes
			// in both cases, they are represented as ExecParent objects
			// (but we include an extra Label field containing the dataset/node name)
			options: [],

			form: {
				// selected type of task
				task: null,
				// name for the ExecNode
				name: '',
				// selected model ID
				modelID: '',
				// selected model object
				model: '',
				// model mode option, only set if Model.Modes is set
				mode: '',
				// use custom/existing model or get a pre-trained one?
				// must be either 'pretrain' or 'custom'
				modelType: 'pretrain',
				// if modelType==pretrain, pretrain on which dataset?
				// refers to Models[..].Pretrain[..].ID
				pretrain: '',
				// if modelType==custom, which file dataset to get the model from?
				// this is an index in options
				customParentIdx: null,
				// input parent (index in this.options)
				inputParentIdx: null,
			},

			// (1) 'form': submitting the form
			// (2) 'importing': running import job for pre-trained model
			// (3) 'error': error setting it up
			phase: 'form',
			// job if phase='importing'
			job: null,
			// if phase='error'
			errorMsg: '',
		};
	},
	created: function() {
		get_parent_options(this.$route.params.ws, this, (options) => {
			this.options = options;
		});
	},
	methods: {
		selectTask: function(task) {
			this.form.task = task;
		},
		changedModel: function() {
			this.form.model = null;
			this.form.modelType = 'pretrain';
			this.form.pretrain = '';
			if(this.form.task.Models[this.form.modelID]) {
				this.form.model = this.form.task.Models[this.form.modelID];
				if(this.form.model.Pretrain) {
					this.form.pretrain = this.form.model.Pretrain[0].ID;
				} else {
					this.form.modelType = 'custom';
				}
			}
			this.form.mode = '';
		},
		addNode: function() {
			let handle = async () => {
				let setError = (errorMsg, e) => {
					console.log(errorMsg, e);
					this.phase = 'error';
					this.errorMsg = errorMsg;
				}

				// get the model ExecParent
				// if modelType=='pretrain', we need to import the pre-trained model
				// if modelType=='custom', it is just customParent
				let modelParent;
				if(this.form.modelType == 'pretrain') {
					// import pre-trained model if needed
					let importParams = {
						mode: 'url',
						url: 'https://skyhookml.org/datasets/'+this.form.model.ID+'-'+this.form.mode+'-'+this.form.pretrain+'.zip',
					};
					let importJob;
					try {
						importJob = await utils.request(this, 'POST', '/import-dataset?mode=url', importParams);
					} catch(e) {
						setError('Error starting import job: ' + e.responseText, e);
						return;
					}
					this.job = importJob;
					this.phase = 'importing';
					console.log('waiting for import job');
					try {
						importJob = await utils.waitForJob(this.job.ID);
					} catch(e) {
						setError('Error importing pre-trained model: ' + e.Error, e);
						return;
					}
					console.log('import job completed, found dataset', importJob.Metadata);
					modelParent = {
						Type: 'd',
						ID: parseInt(importJob.Metadata),
						DataType: 'file',
					};
				} else if(this.form.modelType == 'custom') {
					modelParent = this.options[this.form.customParentIdx];
				}

				// create parents dict
				let parents = {
					'input': [this.options[this.form.inputParentIdx]],
					'model': [modelParent],
				};

				// set node params
				let nodeParams = {};
				if(this.form.mode) {
					nodeParams['Mode'] = this.form.mode;
				}

				let op = this.form.model.ID+'_infer';

				// custom stuff
				if(this.form.model.ID == 'unsupervised_reid') {
					op = 'reid_tracker';
					parents = {
						'model': [modelParent],
						'video': [this.options[this.form.inputParentIdx]],
					};
				}

				// create the node
				let params = {
					Name: this.form.name,
					Op: op,
					Params: JSON.stringify(nodeParams),
					Parents: parents,
					Workspace: this.$route.params.ws,
				};
				let node;
				try {
					node = await utils.request(this, 'POST', '/exec-nodes', JSON.stringify(params));
				} catch(e) {
					setError('Error creating ExecNode: ' + e.responseText, e);
					return;
				}

				this.$router.push('/ws/'+this.$route.params.ws+'/exec/'+node.ID);
			};
			handle();
		},
	},
	template: `
<div class="flex-container">
	<template v-if="phase == 'form'">
		<div class="small-container">
			<h3>Apply a Model</h3>
			<template v-if="!form.task">
				<p>Select a task:</p>
				<template v-for="task in tasks">
					<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectTask(task)">
						<div class="card-body">
							<h5 class="card-title">{{ task.Name }}</h5>
							<p class="card-text">{{ task.Help }}</p>
						</div>
					</div>
				</template>
			</template>
			<template v-else>
				<form v-on:submit.prevent="addNode">
					<div class="row mb-2">
						<label class="col-sm-4 col-form-label">Name</label>
						<div class="col-sm-8">
							<input v-model="form.name" type="text" class="form-control" required />
							<small class="form-text text-muted">A name for this node.</small>
						</div>
					</div>
					<div class="row mb-2">
						<label class="col-sm-4 col-form-label">Model</label>
						<div class="col-sm-8">
							<div class="form-check" v-for="model in form.task.Models">
								<input class="form-check-input" type="radio" v-model="form.modelID" :value="model.ID" @change="changedModel">
								<label class="form-check-label">{{ model.Name }}</label>
							</div>
						</div>
					</div>
					<div v-if="form.model && form.model.Modes" class="row mb-2">
						<label class="col-sm-4 col-form-label">Mode</label>
						<div class="col-sm-8">
							<div class="form-check" v-for="mode in form.model.Modes">
								<input class="form-check-input" type="radio" v-model="form.mode" :value="mode.ID">
								<label class="form-check-label">{{ mode.Name }}</label>
							</div>
						</div>
					</div>
					<div v-if="form.model && form.model.Pretrain" class="row mb-2">
						<label class="col-sm-4 col-form-label">Model Type</label>
						<div class="col-sm-8">
							<select v-model="form.modelType" class="form-select">
								<option value="pretrain">Pre-trained Model</option>
								<option value="custom">Custom Model</option>
							</select>
						</div>
					</div>
					<div v-if="form.modelType == 'pretrain'" class="row mb-2">
						<label class="col-sm-4 col-form-label">Pre-training</label>
						<div class="col-sm-8">
							<select v-model="form.pretrain" class="form-select">
								<template v-for="opt in form.model.Pretrain">
									<option :key="opt.ID" :value="opt.ID">{{ opt.Name }}</option>
								</template>
							</select>
							<small class="form-text text-muted">Select a pre-trained model.</small>
						</div>
					</div>
					<div v-if="form.modelType == 'custom'" class="row mb-2">
						<label class="col-sm-4 col-form-label">Custom Model</label>
						<div class="col-sm-8">
							<select v-model="form.customParentIdx" class="form-select">
								<template v-for="(opt, idx) in options">
									<option v-if="opt.DataType == 'file'" :value="idx">{{ opt.Label }}</option>
								</template>
							</select>
						</div>
					</div>
					<div class="row mb-2">
						<label class="col-sm-4 col-form-label">Input Dataset</label>
						<div class="col-sm-8">
							<select v-model="form.inputParentIdx" class="form-select">
								<template v-for="(opt, idx) in options">
									<option v-if="opt.DataType == 'video' || opt.DataType == 'image'" :value="idx">{{ opt.Label }}</option>
								</template>
							</select>
							<small class="form-text text-muted">The image or video dataset to apply the model on.</small>
						</div>
					</div>
					<div class="row mb-2">
						<div class="col-sm-12">
							<button type="submit" class="btn btn-primary">Add Node</button>
						</div>
					</div>
				</form>
			</template>
		</div>
	</template>
	<template v-else-if="phase == 'importing'">
		<h3>Importing...</h3>
		<div class="flex-content">
			<job-consoleprogress :jobID="job.ID"></job-consoleprogress>
		</div>
	</template>
	<template v-else-if="phase == 'error'">
		<div class="small-container">
			<h3>Error</h3>
			<p>{{ errorMsg }}</p>
		</div>
	</template>
</div>
	`,
};
