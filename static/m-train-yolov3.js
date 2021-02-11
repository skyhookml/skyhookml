import utils from './utils.js';

export default {
	data: function() {
		return {
			node: null,
			params: {
				inputWidth: '',
				inputHeight: '',
				configPath: '',
				imageDatasetID: '',
				detectionDatasetID: '',
			},
			datasets: [],
		};
	},
	created: function() {
		utils.request(this, 'GET', '/datasets', null, (datasets) => {
			this.datasets = datasets;
		});

		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/train-nodes/'+nodeID, null, (node) => {
			this.node = node;
			try {
				let s = JSON.parse(this.node.Params);
				if(s.InputSize) {
					this.params.inputWidth = s.InputSize[0];
					this.params.inputHeight = s.InputSize[1];
				}
				if(s.ConfigPath) {
					this.params.configPath = s.ConfigPath;
				}
				if(s.ImageDatasetID) {
					this.params.imageDatasetID = s.ImageDatasetID;
				}
				if(s.DetectionDatasetID) {
					this.params.detectionDatasetID = s.DetectionDatasetID;
				}
			} catch(e) {}
		});
	},
	methods: {
		save: function() {
			let params = {
				InputSize: [parseInt(this.params.inputWidth), parseInt(this.params.inputHeight)],
				ConfigPath: this.params.configPath,
				ImageDatasetID: parseInt(this.params.imageDatasetID),
				DetectionDatasetID: parseInt(this.params.detectionDatasetID),
			};
			utils.request(this, 'POST', '/train-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}));
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Input Width</label>
			<div class="col-sm-10">
				<input v-model="params.inputWidth" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Input Height</label>
			<div class="col-sm-10">
				<input v-model="params.inputHeight" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Config Path</label>
			<div class="col-sm-10">
				<input v-model="params.configPath" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Image Dataset</label>
			<div class="col-sm-10">
				<select v-model="params.imageDatasetID" class="form-control">
					<template v-for="ds in datasets">
						<option :key="ds.ID" :value="ds.ID">{{ ds.Name }}</option>
					</template>
				</select>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Detection Dataset</label>
			<div class="col-sm-10">
				<select v-model="params.detectionDatasetID" class="form-control">
					<template v-for="ds in datasets">
						<option :key="ds.ID" :value="ds.ID">{{ ds.Name }}</option>
					</template>
				</select>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
