import utils from './utils.js';

export default {
	data: function() {
		return {
			datasets: [],
			form: {
				// selected type of task
				task: null,
				// name for the ExecNode
				name: '',
				// selected model name
				model: '',
				// whether to use pre-training
				pretrain: true,
				// input dataset IDs, which should correspond to task.Inputs
				inputIDs: [],
			},

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
					Models: [{
						ID: 'pytorch_yolov3',
						Name: 'YOLOv3',
					}, {
						ID: 'pytorch_scaled_yolov4',
						Name: 'Scaled-YOLOv4',
					}, {
						ID: 'pytorch_yolov5',
						Name: 'Scaled-YOLOv5',
					}, {
						ID: 'pytorch_maskrcnn',
						Name: 'Mask R-CNN',
					}, {
						ID: 'pytorch_efficientdet',
						Name: 'EfficientDet',
					}, {
						ID: 'pytorch_mobilenetssd',
						Name: 'MobileNet+SSD',
					}],
					Pretrain: 'COCO',
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
					Models: [{
						ID: 'pytorch_resnet34',
						Name: 'Resnet34',
					}, {
						ID: 'pytorch_efficientnet',
						Name: 'EfficientNet',
					}, {
						ID: 'pytorch_mobilenet',
						Name: 'MobileNet',
					}, {
						ID: 'pytorch_vgg',
						Name: 'VGG',
					}],
					Pretrain: 'ImageNet',
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
		addNode: function() {
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
			// create the node
			let params = {
				Name: this.form.name,
				Op: this.form.model+'_train',
				Params: '',
				Parents: parents,
				Workspace: this.$route.params.ws,
			};
			utils.request(this, 'POST', '/exec-nodes', JSON.stringify(params), (node) => {
				this.$router.push('/ws/'+this.$route.params.ws+'/exec/'+node.Op+'/'+node.ID);
			});
		},
	},
	template: `
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
						<input class="form-check-input" type="radio" v-model="form.model" :value="model.ID">
						<label class="form-check-label">{{ model.Name }}</label>
					</div>
				</div>
			</div>
			<div class="row mb-2">
				<label class="col-sm-4 col-form-label">Pre-training</label>
				<div class="col-sm-8">
					<div class="form-check">
						<input class="form-check-input" type="checkbox" v-model="form.pretrain">
						<label class="form-check-label">
							Fine-tune a pre-trained model instead of training from scratch (pre-trained on {{ form.task.Pretrain }})
						</label>
					</div>
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
	`,
};
