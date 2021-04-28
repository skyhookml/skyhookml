<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Architecture</label>
			<div class="col-sm-10">
				<select v-model="params.archID" class="form-select">
					<template v-for="arch in archs">
						<option :key="arch.ID" :value="arch.ID">{{ arch.ID }}</option>
					</template>
				</select>
			</div>
		</div>
		<template v-if="arch">
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
									<select v-model="addForms.inputIdx" class="form-select">
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
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Outputs</label>
				<div class="col-sm-10">
					<table class="table">
						<thead>
							<tr>
								<th>Component</th>
								<th>Layer</th>
								<th>Data Type</th>
								<th></th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="(spec, i) in params.outputDatasets">
								<td>
									<template v-if="getComponent(spec.ComponentIdx)">{{ getComponent(spec.ComponentIdx).ID }}</template>
									<template v-else>Component {{ spec.ComponentIdx }}</template>
								</td>
								<td>{{ spec.Layer }}</td>
								<td>{{ spec.DataType }}</td>
								<td>
									<button type="button" class="btn btn-danger" v-on:click="removeOutput(i)">Remove</button>
								</td>
							</tr>
							<tr>
								<td>
									<select v-model="addForms.outputComponentIdx" class="form-select">
										<template v-for="(compSpec, compIdx) in arch.Params.Components">
											<option v-if="compSpec.ID in comps" :key="compIdx" :value="compIdx">{{ comps[compSpec.ID].ID }}</option>
										</template>
									</select>
								</td>
								<td>
									<template v-if="getComponent(addForms.outputComponentIdx)">
										<select v-model="addForms.outputLayer" class="form-select">
											<template v-for="(_, layer) in getComponent(addForms.outputComponentIdx).Params.Outputs">
												<option :key="layer" :value="layer">{{ layer }}</option>
											</template>
										</select>
									</template>
								</td>
								<td></td>
								<td>
									<button type="button" class="btn btn-primary" v-on:click="addOutput">Add</button>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</template>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			params: {
				archID: '',
				inputOptions: [],
				outputDatasets: [],
			},
			// list of {Name, DataType}
			parents: [],
			archs: {},
			comps: {},
			addForms: null,
		};
	},
	props: ['node'],
	created: function() {
		this.resetForm();

		utils.request(this, 'GET', '/pytorch/archs', null, (archs) => {
			archs.forEach((arch) => {
				this.$set(this.archs, arch.ID, arch);
			});
		});
		utils.request(this, 'GET', '/pytorch/components', null, (comps) => {
			comps.forEach((comp) => {
				this.$set(this.comps, comp.ID, comp);
			});
		});

		try {
			let s = JSON.parse(this.node.Params);
			if(s.ArchID) {
				this.params.archID = s.ArchID;
			}
			if(s.InputOptions) {
				this.params.inputOptions = s.InputOptions;
			}
			if(s.OutputDatasets) {
				this.params.outputDatasets = s.OutputDatasets;
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
		if(this.node.Parents && this.node.Parents['inputs']) {
			inputs = this.node.Parents['inputs'];
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
	},
	methods: {
		resetForm: function() {
			this.addForms = {
				inputIdx: '',
				inputOptions: '',
				outputComponentIdx: '',
				outputLayer: '',
			};
		},
		save: function() {
			let params = {
				ArchID: this.params.archID,
				InputOptions: this.params.inputOptions,
				OutputDatasets: this.params.outputDatasets,
			};
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
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
		addOutput: function() {
			let componentIdx = parseInt(this.addForms.outputComponentIdx);
			let layer = this.addForms.outputLayer;
			this.params.outputDatasets.push({
				ComponentIdx: componentIdx,
				Layer: layer,
				DataType: this.getComponent(componentIdx).Params.Outputs[layer],
			});
			this.resetForm();
		},
		removeOutput: function(i) {
			this.params.outputDatasets.splice(i, 1);
		},
		getComponent: function(compIdx) {
			if(compIdx === '') {
				return null;
			}
			compIdx = parseInt(compIdx);
			if(!this.arch) {
				return null;
			}
			if(compIdx >= this.arch.Params.Components.length) {
				return null;
			}
			let compID = this.arch.Params.Components[compIdx].ID;
			return this.comps[compID];
		},
	},
	computed: {
		arch: function() {
			return this.archs[this.params.archID];
		},
	},
};
</script>