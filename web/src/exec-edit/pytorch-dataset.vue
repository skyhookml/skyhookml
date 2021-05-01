<template>
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
		<template v-if="['image', 'video', 'array'].includes(parent.DataType)">
			<select-input-size v-model="inputOptions[idx]" @change="update"></select-input-size>
		</template>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';
import SelectInputSize from './select-input-size.vue';

export default {
	components: {
		'select-input-size': SelectInputSize,
	},
	data: function() {
		return {
			parents: [],
			inputOptions: [],
			valPercent: 20,
		};
	},
	props: ['node', 'value'],
	created: function() {
		try {
			let s = JSON.parse(this.value);
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

		let inputs = [];
		if(this.node.Parents) {
			inputs = this.node.Parents['inputs'];
		}
		while(this.inputOptions.length < inputs.length) {
			this.inputOptions.push({});
		}
		this.parents = [];
		inputs.forEach((parent, idx) => {
			this.parents.push({
				Name: 'unknown',
				DataType: 'unknown',
			});
			if(parent.Type == 'n') {
				utils.request(this, 'GET', '/exec-nodes/'+parent.ID, null, (node) => {
					this.parents[idx].Name = node.Name;
					let index = getIndexByKeyValue(node.Outputs, 'Name', parent.Name);
					this.parents[idx].DataType = node.Outputs[index].DataType;
				});
			} else if(parent.Type == 'd') {
				utils.request(this, 'GET', '/datasets/'+parent.ID, null, (ds) => {
					this.parents[idx].Name = ds.Name;
					this.parents[idx].DataType = ds.DataType;
				});
			}
		});
		this.update();
	},
	methods: {
		update: function() {
			this.$emit('input', JSON.stringify({
				InputOptions: this.inputOptions,
				ValPercent: this.valPercent,
			}));
		},
	},
};
</script>
