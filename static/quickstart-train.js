import utils from './utils.js';
import JobConsoleProgress from './job-consoleprogress.js';

export default {
	components: {
		'job-consoleprogress': JobConsoleProgress,
	},
	data: function() {
		return {
			datasets: [],
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
				// input dataset IDs, which should correspond to task.Inputs
				inputIDs: [],
			},

			// (1) 'form': submitting the form
			// (2) 'importing': running import job for pre-trained model
			// (3) 'error': error setting it up
			phase: 'form',
			// job if phase='importing'
			job: null,
			// if phase='error'
			errorMsg: '',

			tasks: {
				"detection": {
					Name: "Object Detection",
					Help: "Train a model to detect bounding boxes of instances of one or more object categories in images.",
					Inputs: [{
						ID: "images",
						Name: "Images",
						DataType: "image",
						Help: "An image dataset containing example inputs.",
					}, {
						ID: "detections",
						Name: "Detection Labels",
						DataType: "detection",
						Help: "A detection dataset containing bounding box labels corresponding to each input image.",
					}],
					Models: {
						'pytorch_yolov3': {
							ID: 'pytorch_yolov3',
							Name: 'YOLOv3',
							Modes: [
								{ID: 'yolov3', Name: 'YOLOv3'},
								{ID: 'yolov3-spp', Name: 'YOLOv3-SPP'},
								{ID: 'yolov3-tiny', Name: 'YOLOv3-Tiny'},
							],
							ModeHelp: `
								YOLOv3 and YOLOv3-SPP are large models providing high accuracy (YOLOv3-SPP may provide slightly higher accuarcy).
								YOLOv3-Tiny is a small model that is fast but provides lower accuracy.
							`,
							Pretrain: [{
								ID: 'coco',
								Name: 'COCO',
							}],
						},
						'pytorch_yolov5': {
							ID: 'pytorch_yolov5',
							Name: 'YOLOv5',
							Modes: [
								{ID: 'x', Name: 'YOLOv5x'},
								{ID: 'l', Name: 'YOLOv5l'},
								{ID: 'm', Name: 'YOLOv5m'},
								{ID: 's', Name: 'YOLOv5s'},
							],
							ModeHelp: `
								Larger models like YOLOv5l and YOLOv5x provide greater accuracy but slower inference than smaller models like YOLOv5s.
							`,
							Pretrain: [{
								ID: 'coco',
								Name: 'COCO',
							}],
						},
						'pytorch_mobilenetssd': {
							ID: 'pytorch_mobilenetssd',
							Name: 'MobileNet+SSD',
							Pretrain: [{
								ID: 'voc2007',
								Name: 'VOC 2007',
							}],
						},
					},
				},
				"classification": {
					Name: "Image Classification",
					Help: "Train a model to classify images into categories.",
					Inputs: [{
						ID: "images",
						Name: "Images",
						DataType: "image",
						Help: "An image dataset containing example inputs.",
					}, {
						ID: "labels",
						Name: "Classification Labels",
						DataType: "int",
						Help: "An integer dataset containing category labels corresponding to each input image.",
					}],
					Models: {
						'pytorch_resnet34': {
							ID: 'pytorch_resnet34',
							Name: 'Resnet34',
						},
						'pytorch_efficientnet': {
							ID: 'pytorch_efficientnet',
							Name: 'EfficientNet',
						},
						'pytorch_mobilenet': {
							ID: 'pytorch_mobilenet',
							Name: 'MobileNet',
						},
						'pytorch_vgg': {
							ID: 'pytorch_vgg',
							Name: 'VGG',
						},
					},
				},
			},
		};
	},
	created: function() {
		utils.request(this, 'GET', '/datasets', null, (data) => {
			this.datasets = data;
		});
	},
	methods: {
		selectTask: function(task) {
			this.form.task = task;
			this.form.inputIDs = [];
			for(let i = 0; i < task.Inputs.length; i++) {
				this.form.inputIDs.push(null);
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

				// create ExecParents from dataset inputIDs
				let parents = {};
				for(let [idx, datasetID] of this.form.inputIDs.entries()) {
					let input = this.form.task.Inputs[idx];
					parents[input.ID] = [{
						Type: 'd',
						ID: datasetID,
						DataType: input.DataType,
					}];
				}

				// import pre-trained model if needed
				if(this.form.pretrain) {
					let importParams = {
						mode: 'url',
						url: 'https://favyen.com/files/skyhookml/'+this.form.model.ID+'-'+this.form.mode+'-'+this.form.pretrain+'.zip',
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

				// create the node
				let nodeParams = {};
				if(this.form.mode) {
					nodeParams['Mode'] = this.form.mode;
				}
				nodeParams['Train'] = {
					'Op': 'default',
					'Params': JSON.stringify({
						'BatchSize': 8,
					}),
				}
				if(this.form.model.ID == 'pytorch_yolov3') {
					// exclude last layer which is dependent on # categories
					nodeParams['Restore'] = [{
						'SrcPrefix': '',
						'DstPrefix': '',
						'SkipPrefixes': 'mlist.0.model.model.28.',
					}];
				} else if(this.form.model.ID == 'pytorch_yolov5') {
					// exclude last layer which is dependent on # categories
					nodeParams['Restore'] = [{
						'SrcPrefix': '',
						'DstPrefix': '',
						'SkipPrefixes': 'mlist.0.model.model.24.',
					}];
				}
				let params = {
					Name: this.form.name,
					Op: this.form.model.ID+'_train',
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
							<input v-model="form.name" type="text" class="form-control">
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
					<template v-for="(input, i) in form.task.Inputs">
						<div class="row mb-2">
							<label class="col-sm-4 col-form-label">{{ input.Name }}</label>
							<div class="col-sm-8">
								<select v-model="form.inputIDs[i]" class="form-select">
									<template v-for="ds in datasets">
										<option
											v-if="input.DataType == ds.DataType"
											:key="ds.ID"
											:value="ds.ID">
											{{ ds.Name }}
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
