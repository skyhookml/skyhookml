import utils from './utils.js';
import PytorchAugment from './pytorch-augment.js';
import PytorchRestore from './pytorch-restore.js';
import PytorchTrain from './pytorch-train.js';

export default {
	components: {
		'pytorch-augment': PytorchAugment,
		'pytorch-restore': PytorchRestore,
		'pytorch-train': PytorchTrain,
	},
	data: function() {
		return {
			params: null,
		};
	},
	props: ['node'],
	created: function() {
		let params = {};
		try {
			params = JSON.parse(this.node.Params);
		} catch(e) {}
		if(!('Augment' in params)) params.Augment = [];
		if(!('Train' in params)) {
			params.Train = {
				Op: 'default',
				Params: '',
			};
		}
		if(!('Restore' in params)) params.Restore = [];
		if(!('Mode' in params)) params.Mode = 'yolov3';
		if(!('ValPercent' in params)) params.ValPercent = 20;
		this.params = params;
	},
	methods: {
		save: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(this.params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/queries');
			});
		},
	},
	template: `
<div class="m-2">
	<template v-if="node != null">
		<ul class="nav nav-tabs" id="m-nav" role="tablist">
			<li class="nav-item">
				<a class="nav-link active" id="pytorch-model-tab" data-toggle="tab" href="#pytorch-basic-panel" role="tab">Basic</a>
			</li>
			<li class="nav-item">
				<a class="nav-link" id="pytorch-augment-tab" data-toggle="tab" href="#pytorch-augment-panel" role="tab">Data Augmentation</a>
			</li>
			<li class="nav-item">
				<a class="nav-link" id="pytorch-restore-tab" data-toggle="tab" href="#pytorch-restore-panel" role="tab">Restore</a>
			</li>
			<li class="nav-item">
				<a class="nav-link" id="pytorch-train-tab" data-toggle="tab" href="#pytorch-train-panel" role="tab">Training</a>
			</li>
		</ul>
		<div class="tab-content mx-1">
			<div class="tab-pane fade show active" id="pytorch-basic-panel" role="tabpanel">
				<div class="small-container">
					<div class="row mb-2">
						<label class="col-sm-4 col-form-label">Mode</label>
						<div class="col-sm-8">
							<select v-model="params.Mode" class="form-select">
								<option value="vgg16-ssd">VGG+SSD</option>
								<option value="mb1-ssd">MobileNetv1+SSD</option>
								<option value="mb1-ssd-lite">MobileNetv1+SSD-Lite</option>
								<option value="sq-ssd-lite">SqueezeNet+SSD-Lite</option>
								<option value="mb2-ssd-lite">MobileNetv2+SSD-Lite</option>
								<option value="mb3-large-ssd-lite">MobileNetv3-Large+SSD-Lite</option>
								<option value="mb3-small-ssd-lite">MobileNetv3-Small+SSD-Lite</option>
							</select>
						</div>
					</div>
					<div class="row mb-2">
						<label class="col-sm-4 col-form-label">Validation Percentage</label>
						<div class="col-sm-8">
							<input v-model.number="params.ValPercent" type="text" class="form-control">
							<small class="form-text text-muted">
								Use this percentage of the input data for validation. The rest will be used for training.
							</small>
						</div>
					</div>
				</div>
			</div>
			<div class="tab-pane fade" id="pytorch-augment-panel" role="tabpanel">
				<pytorch-augment v-bind:node="node" v-model="params.Augment"></pytorch-augment>
			</div>
			<div class="tab-pane fade" id="pytorch-restore-panel" role="tabpanel">
				<pytorch-restore v-bind:node="node" v-model="params.Restore"></pytorch-restore>
			</div>
			<div class="tab-pane fade" id="pytorch-train-panel" role="tabpanel">
				<pytorch-train v-bind:node="node" v-model="params.Train.Params"></pytorch-train>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
