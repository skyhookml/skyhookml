import utils from './utils.js';

export default {
	data: function() {
		return {
			parents: [],
			inputOptions: [],
			valPercent: 20,
		};
	},
	props: ['node', 'params'],
	created: function() {
		try {
			let s = JSON.parse(this.params);
			if(s.InputOptions) {
				this.inputOptions = s.InputOptions;
			}
			if(s.ValPercent) {
				this.valPercent = s.ValPercent;
			}
		} catch(e) {}

		// fetch parents to get details on the datasets for which the user can configure input options

		// given an array of objects, get index of the object in the array
		// that has a certain value at some key
		let getIndexByKeyValue = function(array, key, value) {
			let index = -1;
			array.forEach((obj, i) => {
				if(obj[key] != value) {
					return;
				}
				index = i;
			});
			return index;
		};

		// after figuring out the data type at input idx, add missing options for that type to inputOptions
		let addMissingOptions = function(idx) {
			let dtype = this.parents[idx].DataType;
			if(dtype == 'image') {
				if('Width' in this.inputOptions[idx] && 'Height' in this.inputOptions[idx]) {
					return;
				}
				this.$set(this.inputOptions, idx, {
					Width: 0,
					Height: 0,
				});
			}
		}

		let inputs = [];
		if(this.node.Parents) {
			// get Parents['inputs'], but Parents is array
			// really the index should always be 0 for pytorch_train node
			let index = getIndexByKeyValue(this.node.Inputs, 'Name', 'inputs');
			if(index >= 0) {
				inputs = this.node.Parents[index];
			}
		}
		this.parents = [];
		inputs.forEach((parent, idx) => {
			this.parents.push({
				Name: 'unknown',
				DataType: 'unknown',
			});
			if(!this.inputOptions[idx]) {
				this.$set(this.inputOptions, idx, {});
			}
			if(parent.Type == 'n') {
				utils.request(this, 'GET', '/exec-nodes/'+parent.ID, null, (node) => {
					this.parents[idx].Name = node.Name;
					let index = getIndexByKeyValue(node.Outputs, 'Name', parent.Name);
					this.parents[idx].DataType = node.Outputs[index].DataType;
					this.addMissingOptions(idx);
				});
			} else if(parent.Type == 'd') {
				utils.request(this, 'GET', '/datasets/'+parent.ID, null, (ds) => {
					this.parents[idx].Name = ds.Name;
					this.parents[idx].DataType = ds.DataType;
					this.addMissingOptions(idx);
				});
			}
		});
	},
	methods: {
		update: function() {
			this.$emit('input', JSON.stringify({
				InputOptions: this.inputOptions,
				valPercent: this.valPercent,
			}));
		},
	},
	template: `
<div class="small-container">
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Validation Percentage</label>
		<div class="col-sm-10">
			<input v-model.number="valPercent" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Use this percentage of the input data for validation. The rest will be used for training.
			</small>
		</div>
	</div>
	<h3>Input Options</h3>
	<template v-for="(parent, idx) in parents">
		<h4>{{ parent.Name }} ({{ parent.DataType }})</h4>
		<template v-if="parent.DataType == 'image'">
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Width</label>
				<div class="col-sm-10">
					<input v-model.number="inputOptions[idx].Width" type="text" class="form-control" @change="update">
					<small class="form-text text-muted">
						Resize the image to this width. Leave as 0 to use the input image without resizing.
					</small>
				</div>
			</div>
				<div class="form-group row">
					<label class="col-sm-2 col-form-label">Height</label>
					<div class="col-sm-10">
						<input v-model.number="inputOptions[idx].Height" type="text" class="form-control" @change="update">
						<small class="form-text text-muted">
							Resize the image to this height. Leave as 0 to use the input image without resizing.
						</small>
					</div>
				</div>
		</template>
	</template>
</div>
	`,
};
