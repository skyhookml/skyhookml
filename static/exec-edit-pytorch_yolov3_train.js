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
			node: null,
			params: null,
		};
	},
	created: function() {
		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/exec-nodes/'+nodeID, null, (node) => {
			this.node = node;
			let params = {};
			try {
				params = JSON.parse(this.node.Params);
			} catch(e) {}
			if(!params.Augment) {
				params.Augment = [];
			}
			if(!params.Train) {
				params.Train = {
					Op: 'default',
					Params: '',
				};
			}
			if(!params.Restore) {
				params.Restore = [];
			}
			if(!params.Width) {
				params.Width = 0;
			}
			if(!params.Height) {
				params.Height = 0;
			}
			if(!params.ValPercent) {
				params.ValPercent = 20;
			}
			this.params = params;
		});
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
						<label class="col-sm-4 col-form-label">Width</label>
						<div class="col-sm-8">
							<input v-model.number="params.Width" type="text" class="form-control">
							<small class="form-text text-muted">
								Resize the image to this width. Leave as 0 to use the input image without resizing.
							</small>
						</div>
					</div>
					<div class="row mb-2">
						<label class="col-sm-4 col-form-label">Height</label>
						<div class="col-sm-8">
							<input v-model.number="params.Height" type="text" class="form-control">
							<small class="form-text text-muted">
								Resize the image to this height. Leave as 0 to use the input image without resizing.
							</small>
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
