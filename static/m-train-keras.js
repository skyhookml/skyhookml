Vue.component('m-train-keras', {
	data: function() {
		return {
			params: {
				archs: [],
				inputs: [],
				outputs: [],
			},
			datasets: {},
			archs: {},
			addForms: null,
		};
	},
	props: ['node'],
	created: function() {
		this.resetForm();

		myCall('GET', '/keras/archs', null, (archs) => {
			archs.forEach((arch) => {
				this.$set(this.archs, arch.ID, arch);
			});
		});
		myCall('GET', '/datasets', null, (datasets) => {
			datasets.forEach((ds) => {
				this.$set(this.datasets, ds.ID, ds);
			});
		});

		try {
			let s = JSON.parse(this.node.Params);
			if(s.InputDatasets) {
				this.params.inputs = s.InputDatasets;
			}
			if(s.OutputDatasets) {
				this.params.outputs = s.OutputDatasets;
			}
			if(s.Archs) {
				s.Archs.forEach((spec) => {
					this.params.archs.push({
						ID: spec.ID,
						Inputs: spec.Inputs.join(','),
						Outputs: spec.Outputs.join(','),
					});
				});
			}
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = {
				Archs: this.params.archs.map((spec) => {
					return {
						ID: spec.ID,
						Inputs: spec.Inputs.split(',').filter((part) => part != ''),
						Outputs: spec.Outputs.split(',').filter((part) => part != ''),
					};
				}),
				InputDatasets: this.params.inputs.map((s) => parseInt(s)),
				OutputDatasets: this.params.outputs.map((s) => parseInt(s)),
			};
			myCall('POST', '/train-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}));
		},
		resetForm: function() {
			this.addForms = {
				archID: '',
				archInputs: '',
				archOutputs: '',
				inputID: '',
				outputID: '',
			};
		},
		addArch: function() {
			this.params.archs.push({
				ID: parseInt(this.addForms.archID),
				Inputs: this.addForms.archInputs,
				Outputs: this.addForms.archOutputs,
			});
			this.resetForm();
		},
		removeArch: function(i) {
			this.params.archs.splice(i, 1);
		},
		addInput: function() {
			this.params.inputs.push(parseInt(this.addForms.inputID));
			this.resetForm();
		},
		removeInput: function(i) {
			this.params.inputs.splice(i, 1);
		},
		addOutput: function() {
			this.params.outputs.push(parseInt(this.addForms.outputID));
			this.resetForm();
		},
		removeOutput: function(i) {
			this.params.outputs.splice(i, 1);
		},
	},
	template: `
<div class="small-container m-2">
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Architectures</label>
		<div class="col-sm-10">
			<table class="table">
				<tbody>
					<tr v-for="(spec, i) in params.archs">
						<td>
							<template v-if="spec.ID in archs">{{ archs[spec.ID].Name }}</template>
							<template v-else>Unknown</template>
						</td>
						<td>{{ spec.Inputs }}</td>
						<td>{{ spec.Outputs }}</td>
						<td>
							<button type="button" class="btn btn-danger" v-on:click="removeArch(i)">Remove</button>
						</td>
					</tr>
					<tr>
						<td>
							<select v-model="addForms.archID" class="form-control">
								<template v-for="arch in archs">
									<option :key="arch.ID" :value="arch.ID">{{ arch.Name }}</option>
								</template>
							</select>
						</td>
						<td>
							<input v-model="addForms.archInputs" type="text" class="form-control">
						</td>
						<td>
							<input v-model="addForms.archOutputs" type="text" class="form-control">
						</td>
						<td>
							<button type="button" class="btn btn-primary" v-on:click="addArch">Add</button>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Inputs</label>
		<div class="col-sm-10">
			<table class="table">
				<tbody>
					<tr v-for="(dsID, i) in params.inputs">
						<td>
							<template v-if="dsID in datasets">{{ datasets[dsID].Name }}</template>
							<template v-else>Unknown</template>
						</td>
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
				<tbody>
					<tr v-for="(dsID, i) in params.outputs">
						<td>
							<template v-if="dsID in datasets">{{ datasets[dsID].Name }}</template>
							<template v-else>Unknown</template>
						</td>
						<td>
							<button type="button" class="btn btn-danger" v-on:click="removeOutput(i)">Remove</button>
						</td>
					</tr>
					<tr>
						<td>
							<select v-model="addForms.outputID" class="form-control">
								<template v-for="ds in datasets">
									<option :key="ds.ID" :value="ds.ID">{{ ds.Name }}</option>
								</template>
							</select>
						</td>
						<td>
							<button type="button" class="btn btn-primary" v-on:click="addOutput">Add</button>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
	<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
</div>
	`,
});
