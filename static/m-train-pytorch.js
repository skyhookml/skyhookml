import utils from './utils.js';

export default {
	data: function() {
		return {
			node: null,
			params: {
				archID: '',
				inputDatasets: [],
				outputDatasets: [],
			},
			datasets: {},
			archs: {},
			comps: {},
			addForms: null,
		};
	},
	created: function() {
		this.resetForm();

		utils.request(this, 'GET', '/datasets', null, (datasets) => {
			datasets.forEach((ds) => {
				this.$set(this.datasets, ds.ID, ds);
			});
		});
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

		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/train-nodes/'+nodeID, null, (node) => {
			this.node = node;
			try {
				let s = JSON.parse(this.node.Params);
				if(s.ArchID) {
					this.params.archID = s.ArchID;
				}
				if(s.InputDatasets) {
					this.params.inputDatasets = s.InputDatasets;
				}
				if(s.OutputDatasets) {
					this.params.outputDatasets = s.OutputDatasets;
				}
			} catch(e) {}
		});
	},
	methods: {
		resetForm: function() {
			this.addForms = {
				inputID: '',
				inputOptions: '',
				outputComponentIdx: '',
				outputLayer: '',
			};
		},
		save: function() {
			let params = {
				ArchID: parseInt(this.params.archID),
				InputDatasets: this.params.inputDatasets,
				OutputDatasets: this.params.outputDatasets,
			};
			utils.request(this, 'POST', '/train-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}));
		},
		addInput: function() {
			this.params.inputDatasets.push({
				ID: parseInt(this.addForms.inputID),
				Options: this.addForms.inputOptions,
			});
			this.resetForm();
		},
		removeInput: function(i) {
			this.params.inputDatasets.splice(i, 1);
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
		<template v-if="arch">
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Input Datasets</label>
				<div class="col-sm-10">
					<table class="table">
						<tbody>
							<tr v-for="(dsSpec, i) in params.inputDatasets">
								<td>
									<template v-if="dsSpec.ID in datasets">{{ datasets[dsSpec.ID].Name }}</template>
									<template v-else>Unknown</template>
								</td>
								<td>{{ dsSpec.Options }}</td>
								<td>
									<button type="button" class="btn btn-danger" v-on:click="removeInput(i)">Remove</button>
								</td>
							</tr>
							<tr>
								<td>
									<select v-model="addForms.inputID" class="form-control">
										<template v-for="ds in datasets">
											<option :key="ds.ID" :value="ds.ID">{{ ds.Name }}</option>
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
									<template v-if="getComponent(spec.ComponentIdx)">{{ getComponent(spec.ComponentIdx).Name }}</template>
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
									<select v-model="addForms.outputComponentIdx" class="form-control">
										<template v-for="(compSpec, compIdx) in arch.Params.Components">
											<option v-if="compSpec.ID in comps" :key="compIdx" :value="compIdx">{{ comps[compSpec.ID].Name }}</option>
										</template>
									</select>
								</td>
								<td>
									<template v-if="getComponent(addForms.outputComponentIdx)">
										<select v-model="addForms.outputLayer" class="form-control">
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
	`,
};
