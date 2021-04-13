import utils from '../utils.js';
import PytorchModel from './pytorch-model.js';
import PytorchDataset from './pytorch-dataset.js';
import PytorchAugment from './pytorch-augment.js';
import PytorchRestore from './pytorch-restore.js';
import PytorchTrain from './pytorch-train.js';

// Provide an exec-edit component with modules for pytorch training.
// These modules can be optionally disabled.
// Options consist of several fields, all of which are optional:
// - opts.disabled: list of modules to disable
// - opts.basicTemplate: if set, add a Basic tab and insert this template into that tab content
// - opts.created: function to call on created
export default function(opts) {
	let component = {
		components: {
			'pytorch-model': PytorchModel,
			'pytorch-dataset': PytorchDataset,
			'pytorch-augment': PytorchAugment,
			'pytorch-restore': PytorchRestore,
			'pytorch-train': PytorchTrain,
		},
		data: function() {
			return {
				params: null,
				opts: {
					disabled: opts.disabled ? opts.disabled : [],
					enableBasic: 'basicTemplate' in opts,
				},
			};
		},
		props: ['node'],
		created: function() {
			// decode params or initialize defaults
			let params = {};
			try {
				params = JSON.parse(this.node.Params);
			} catch(e) {}
			if(!params.ArchID) {
				params.ArchID = '';
			}
			if(!params.Dataset) {
				params.Dataset = {
					Op: 'default',
					Params: '',
				};
			}
			if(!params.Augment) {
				params.Augment = [];
			}
			if(!params.Restore) {
				params.Restore = [];
			}
			if(!params.Train) {
				params.Train = {
					Op: 'default',
					Params: '',
				};
			}
			this.params = params;
			if(opts.created) {
				opts.created.call(this);
			}
		},
		mounted: function() {
			// show the first tab
			let firstTab = document.querySelector('#m-nav li:first-child button');
			new bootstrap.Tab(firstTab).show();
		},
		methods: {
			save: function() {
				utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
					Params: JSON.stringify(this.params),
				}), () => {
					this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
				});
			},
		},
		template: `
	<div class="m-2">
		<template v-if="node != null">
			<ul class="nav nav-tabs" id="m-nav" role="tablist">
				<li class="nav-item" v-if="opts.enableBasic">
					<button class="nav-link" id="pytorch-model-tab" data-bs-toggle="tab" data-bs-target="#pytorch-basic-panel" role="tab">Basic</button>
				</li>
				<li class="nav-item" v-if="!opts.disabled.includes('model')">
					<button class="nav-link" id="pytorch-model-tab" data-bs-toggle="tab" data-bs-target="#pytorch-model-panel" role="tab">Model</button>
				</li>
				<li class="nav-item" v-if="!opts.disabled.includes('dataset')">
					<button class="nav-link" id="pytorch-dataset-tab" data-bs-toggle="tab" data-bs-target="#pytorch-dataset-panel" role="tab">Datasets</button>
				</li>
				<li class="nav-item" v-if="!opts.disabled.includes('augment')">
					<button class="nav-link" id="pytorch-augment-tab" data-bs-toggle="tab" data-bs-target="#pytorch-augment-panel" role="tab">Data Augmentation</button>
				</li>
				<li class="nav-item" v-if="!opts.disabled.includes('restore')">
					<button class="nav-link" id="pytorch-restore-tab" data-bs-toggle="tab" data-bs-target="#pytorch-restore-panel" role="tab">Restore</button>
				</li>
				<li class="nav-item" v-if="!opts.disabled.includes('train')">
					<button class="nav-link" id="pytorch-train-tab" data-bs-toggle="tab" data-bs-target="#pytorch-train-panel" role="tab">Training</button>
				</li>
			</ul>
			<div class="tab-content mx-1">
				<div v-if="opts.enableBasic" class="tab-pane fade" id="pytorch-basic-panel" role="tabpanel">
					BASIC_TEMPLATE
				</div>
				<div v-if="!opts.disabled.includes('model')" class="tab-pane fade" id="pytorch-model-panel" role="tabpanel">
					<pytorch-model v-bind:node="node" v-model="params.ArchID"></pytorch-model>
				</div>
				<div v-if="!opts.disabled.includes('dataset')" class="tab-pane fade" id="pytorch-dataset-panel" role="tabpanel">
					<pytorch-dataset v-bind:node="node" v-model="params.Dataset.Params"></pytorch-dataset>
				</div>
				<div v-if="!opts.disabled.includes('augment')" class="tab-pane fade" id="pytorch-augment-panel" role="tabpanel">
					<pytorch-augment v-bind:node="node" v-model="params.Augment"></pytorch-augment>
				</div>
				<div v-if="!opts.disabled.includes('restore')" class="tab-pane fade" id="pytorch-restore-panel" role="tabpanel">
					<pytorch-restore v-bind:node="node" v-model="params.Restore"></pytorch-restore>
				</div>
				<div v-if="!opts.disabled.includes('train')" class="tab-pane fade" id="pytorch-train-panel" role="tabpanel">
					<pytorch-train v-bind:node="node" v-model="params.Train.Params"></pytorch-train>
				</div>
			</div>
			<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
		</template>
	</div>
		`,
	};
	if(opts.basicTemplate) {
		component.template = component.template.replace('BASIC_TEMPLATE', opts.basicTemplate);
	}
	return component;
};
