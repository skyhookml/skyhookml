import utils from './utils.js';

export default {
	data: function() {
		return {
			node: null,
			params: {
				archID: '',
				inputOptions: [],
			},
			// list of {Name, DataType}
			parents: [],
			archs: {},
			addForms: null,
		};
	},
	created: function() {
		this.resetForm();

		utils.request(this, 'GET', '/pytorch/archs', null, (archs) => {
			archs.forEach((arch) => {
				this.$set(this.archs, arch.ID, arch);
			});
		});

		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/exec-nodes/'+nodeID, null, (node) => {
			this.node = node;
			try {
				let s = JSON.parse(this.node.Params);
				if(s.ArchID) {
					this.params.archID = s.ArchID;
				}
				if(s.InputOptions) {
					this.params.inputOptions = s.InputOptions;
				}
			} catch(e) {}

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
		});
	},
	methods: {
		resetForm: function() {
			this.addForms = {
				inputIdx: '',
				inputOptions: '',
			};
		},
		save: function() {
			let params = {
				ArchID: parseInt(this.params.archID),
				InputOptions: this.params.inputOptions,
			};
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}));
		},
		addInput: function() {
			this.params.inputOptions.push({
				Idx: parseInt(this.addForms.inputIdx),
				Value: this.addForms.inputOptions,
			});
			this.resetForm();
		},
		removeInput: function(i) {
			this.params.inputOptions.splice(i, 1);
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Architecture</label>
			<div class="col-sm-10">
				<select v-model="params.archID" class="form-control">
					<template v-for="arch in archs">
						<option :key="arch.ID" :value="arch.ID">{{ arch.Name }}</option>
					</template>
				</select>
			</div>
		</div>
		<div class="form-group row" v-if="parents.length > 0">
			<label class="col-sm-2 col-form-label">Input Options</label>
			<div class="col-sm-10">
				<table class="table">
					<tbody>
						<tr v-for="(spec, i) in params.inputOptions">
							<td>{{ parents[spec.Idx].Name }}</td>
							<td>{{ spec.Value }}</td>
							<td>
								<button type="button" class="btn btn-danger" v-on:click="removeInput(i)">Remove</button>
							</td>
						</tr>
						<tr>
							<td>
								<select v-model="addForms.inputIdx" class="form-control">
									<template v-for="(parent, parentIdx) in parents">
										<option :value="parentIdx">{{ parent.Name }} ({{ parent.DataType }})</option>
									</template>
								</select>
							</td>
							<td>
								<input class="form-control" type="text" v-model="addForms.inputOptions" />
							</td>
							<td>
								<button type="button" class="btn btn-primary" v-on:click="addInput">Add</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
