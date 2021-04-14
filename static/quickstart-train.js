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
				// whether to use pre-training, and if so, which dataset?
				// refers to Models[..].Pretrain[..].ID
				pretrain: '',
				// input parents indexes, which should correspond to task.Inputs
				// each element is an index in options
				inputParentIdxs: [],
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
			this.form.inputParentIdxs = [];
			for(let i = 0; i < task.Inputs.length; i++) {
				this.form.inputParentIdxs.push(null);
			}
		},
		changedModel: function() {
			this.form.model = null;
			if(this.form.task.Models[this.form.modelID]) {
				this.form.model = this.form.task.Models[this.form.modelID];
			}
			this.form.mode = '';
			this.form.pretrain = '';
		},
		addNode: function() {
			let handle = async () => {
				let setError = (errorMsg, e) => {
					console.log(errorMsg, e);
					this.phase = 'error';
					this.errorMsg = errorMsg;
				}

				// get ExecParents corresponding to inputParentIdxs
				let parents = {};
				for(let [inputIdx, optionIdx] of this.form.inputParentIdxs.entries()) {
					let input = this.form.task.Inputs[inputIdx];
					parents[input.ID] = [this.options[optionIdx]];
				}

				// import pre-trained model if needed
				if(this.form.pretrain) {
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

					// update parents
					// the dataset ID created by import should be set in job metadata
					parents['models'] = [{
						Type: 'd',
						ID: parseInt(importJob.Metadata),
						DataType: 'file',
					}]
				}

				// set node params
				let nodeParams = {};
				if(this.form.mode) {
					nodeParams['Mode'] = this.form.mode;
				}
				// batch size
				nodeParams['Train'] = {
					'Op': 'default',
					'Params': JSON.stringify({
						'BatchSize': 8,
					}),
				}
				// if pre-train, set SkipPrefixes as necessary
				if(this.form.pretrain) {
					nodeParams['Restore'] = [{
						'SrcPrefix': '',
						'DstPrefix': '',
						'SkipPrefixes': '',
					}];
					// exclude last layer which is dependent on # categories
					if(this.form.model.ID == 'pytorch_yolov3') {
						nodeParams['Restore'][0]['SkipPrefixes'] = 'mlist.0.model.model.28.';
					} else if(this.form.model.ID == 'pytorch_yolov5') {
						nodeParams['Restore'][0]['SkipPrefixes'] = 'mlist.0.model.model.24.';
					} else if(this.form.model.ID == 'pytorch_resnet') {
						nodeParams['Restore'][0]['SkipPrefixes'] = 'mlist.0.model.fc.';
					} else if(this.form.model.ID == 'pytorch_ssd') {
						nodeParams['Restore'][0]['SkipPrefixes'] = 'mlist.0.model.classification_headers.';
					}
				}

				let op = this.form.model.ID+'_train';

				// some custom settings
				if(this.form.model.ID == 'unsupervised_reid') {
					nodeParams['ArchID'] = 'reid';
					op = 'unsupervised_reid';
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
	computed: {
		curInputs: function() {
			if(this.form.model && this.form.model.Inputs) {
				return this.form.model.Inputs;
			} else if(this.form.task) {
				return this.form.task.Inputs;
			} else {
				return [];
			}
		},
	},
	template: `
<div class="flex-container">
	<template v-if="phase == 'form'">
		<div class="small-container">
			<h3>Train a Model</h3>
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
						<label class="col-sm-4 col-form-label">Pre-training</label>
						<div class="col-sm-8">
							<select v-model="form.pretrain" class="form-select">
								<option value="">None</option>
								<template v-for="opt in form.model.Pretrain">
									<option :key="opt.ID" :value="opt.ID">{{ opt.Name }}</option>
								</template>
							</select>
							<small class="form-text text-muted">Fine-tune a pre-trained model instead of training from scratch.</small>
						</div>
					</div>
					<template v-for="(input, i) in curInputs">
						<div class="row mb-2">
							<label class="col-sm-4 col-form-label">{{ input.Name }}</label>
							<div class="col-sm-8">
								<select v-model="form.inputParentIdxs[i]" class="form-select">
									<template v-for="(opt, idx) in options">
										<option
											v-if="input.DataType == opt.DataType"
											:value="idx">
											{{ opt.Label }}
										</option>
									</template>
								</select>
								<small v-if="input.Help" class="form-text text-muted">{{ input.Help }}</small>
							</div>
						</div>
					</template>
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
