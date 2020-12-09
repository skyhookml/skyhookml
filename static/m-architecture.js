Vue.component('m-architecture', {
	data: function() {
		return {
			numInputs: '',
			numTargets: '',
			components: [],
			losses: [],
			addForms: null,

			// parameters for m-architecture-input-modal
			addInputModal: null,

			comps: {},
		};
	},
	props: ['arch'],
	created: function() {
		this.resetForm();

		myCall('GET', '/pytorch/components', null, (comps) => {
			comps.forEach((comp) => {
				this.$set(this.comps, comp.ID, comp);
			});
		});

		var params = this.arch.Params;
		this.numInputs = params.NumInputs;
		this.numTargets = params.NumTargets;
		if(params.Components) {
			this.components = params.Components;
		}
		if(params.Losses) {
			this.losses = params.Losses;
		}
	},
	methods: {
		resetForm: function() {
			this.addForms = {
				componentID: '',
				lossComponentIdx: '',
				lossLayer: '',
				lossWeight: 1.0,
			};
		},
		save: function() {
			let params = {
				NumInputs: parseInt(this.numInputs),
				NumTargets: parseInt(this.numTargets),
				Components: this.components,
				Losses: this.losses,
			};
			myCall('POST', '/pytorch/archs/'+this.arch.ID, JSON.stringify({
				Params: params,
			}));
		},
		addComponent: function() {
			this.components.push({
				ID: parseInt(this.addForms.componentID),
				Params: '',
				Inputs: [],
				Targets: [],
			});
			this.resetForm();
		},
		removeComponent: function(i) {
			this.components.splice(i, 1);
		},
		addLoss: function() {
			this.losses.push({
				ComponentIdx: parseInt(this.addForms.lossComponentIdx),
				Layer: this.addForms.lossLayer,
				Weight: parseFloat(this.addForms.lossWeight),
			});
			this.resetForm();
		},
		removeLoss: function(i) {
			this.losses.splice(i, 1);
		},

		showAddInputModal: function(compIdx, mode) {
			let modalSpec = {
				componentIdx: compIdx,
				mode: mode,
			};

			if(this.addInputModal) {
				this.addInputModal = null;
				Vue.nextTick(() => {
					this.addInputModal = modalSpec;
				});
			} else {
				this.addInputModal = modalSpec;
			}
		},

		// return from m-architecture-input-modal
		addInput: function(e) {
			let compSpec = this.components[this.addInputModal.componentIdx];
			if(this.addInputModal.mode == 'inputs') {
				compSpec.Inputs.push(e);
			} else if(this.addInputModal.mode == 'targets') {
				compSpec.Targets.push(e);
			}

			this.addInputModal = null;
		},

		removeInput: function(compIdx, i) {
			this.components[compIdx].Inputs.splice(i, 1);
		},
		removeTarget: function(compIdx, i) {
			this.components[compIdx].Targets.splice(i, 1);
		},

		parentComponentList: function(compIdx) {
			let comps = [];
			this.components.slice(0, compIdx).forEach((compSpec) => {
				comps.push(this.comps[compSpec.ID]);
			});
			return comps;
		},
		getComponent: function(compIdx) {
			if(compIdx === '') {
				return null;
			}
			compIdx = parseInt(compIdx);
			if(compIdx >= this.components.length) {
				return null;
			}
			let compID = this.components[compIdx].ID;
			return this.comps[compID];
		},
	},
	template: `
<div class="small-container m-2">
	<div class="form-group row">
		<label class="col-sm-2 col-form-label"># Inputs</label>
		<div class="col-sm-10">
			<input v-model="numInputs" type="text" class="form-control">
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label"># Targets</label>
		<div class="col-sm-10">
			<input v-model="numTargets" type="text" class="form-control">
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Components</label>
		<div class="col-sm-10">
			<template v-for="(compSpec, compIdx) in components">
				<h3>Component #{{ compIdx }}<template v-if="compSpec.ID in comps">: {{ comps[compSpec.ID].Name }}</template></h3>
				<p>Inputs:</p>
				<table class="table">
					<tbody>
						<tr v-for="(inp, i) in compSpec.Inputs">
							<td>{{ inp.Type }}</td>
							<td>
								<template v-if="inp.Type == 'dataset'">Dataset {{ inp.DatasetIdx }}</template>
								<template v-else-if="inp.Type == 'layer'">Component #{{ inp.ComponentIdx }} / Layer {{ inp.Layer }}</template>
							</td>
							<td>
								<button type="button" class="btn btn-danger" v-on:click="removeInput(compIdx, i)">Remove</button>
							</td>
						</tr>
					</tbody>
				</table>
				<button type="button" class="btn btn-primary" v-on:click="showAddInputModal(compIdx, 'inputs')">Add Input</button>
				<p>Targets:</p>
				<table class="table">
					<tbody>
						<tr v-for="(inp, i) in compSpec.Targets">
							<td>{{ inp.Type }}</td>
							<td>
								<template v-if="inp.Type == 'dataset'">Dataset {{ inp.DatasetIdx }}</template>
								<template v-else-if="inp.Type == 'layer'">Component #{{ inp.ComponentIdx }} / Layer {{ inp.Layer }}</template>
							</td>
							<td>
								<button type="button" class="btn btn-danger" v-on:click="removeTarget(compIdx, i)">Remove</button>
							</td>
						</tr>
					</tbody>
				</table>
				<button type="button" class="btn btn-primary" v-on:click="showAddInputModal(compIdx, 'targets')">Add Target</button>
				<p>Parameters:</p>
				<textarea v-model="compSpec.Params" class="form-control" rows="5"></textarea>
			</template>
			<hr />
			<form class="form-inline" v-on:submit.prevent="addComponent">
				<select class="form-control my-1 mx-1" v-model="addForms.componentID">
					<option v-for="comp in comps" :key="comp.ID" :value="comp.ID">{{ comp.Name }}</option>
				</select>
				<button type="submit" class="btn btn-primary my-1 mx-1">Add Component</button>
			</form>
		</div>
	</div>
	<template v-if="addInputModal">
		<m-architecture-input-modal v-bind:components="parentComponentList(addInputModal.componentIdx)" v-on:success="addInput($event)"></m-architecture-input-modal>
	</template>
	<div class="form-group row">
		<label class="col-sm-2 col-form-label">Losses</label>
		<div class="col-sm-10">
			<table class="table">
				<thead>
					<tr>
						<th>Component</th>
						<th>Layer</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="(spec, i) in losses">
						<td>
							Component #{{ spec.ComponentIdx }}<template v-if="getComponent(spec.ComponentIdx)">: {{ getComponent(spec.ComponentIdx).Name }}</template>
						</td>
						<td>{{ spec.Layer }}</td>
						<td>{{ spec.Weight }}</td>
						<td>
							<button type="button" class="btn btn-danger" v-on:click="removeLoss(i)">Remove</button>
						</td>
					</tr>
					<tr>
						<td>
							<select v-model="addForms.lossComponentIdx" class="form-control">
								<template v-for="(compSpec, compIdx) in components">
									<option v-if="compSpec.ID in comps" :key="compIdx" :value="compIdx">Component #{{ compIdx }}: {{ comps[compSpec.ID].Name }}</option>
								</template>
							</select>
						</td>
						<td>
							<template v-if="getComponent(addForms.lossComponentIdx)">
								<select v-model="addForms.lossLayer" class="form-control">
									<template v-for="layer in getComponent(addForms.lossComponentIdx).Params.Losses">
										<option :key="layer" :value="layer">{{ layer }}</option>
									</template>
								</select>
							</template>
						</td>
						<td>
							<input class="form-control" type="text" v-model="addForms.lossWeight" />
						</td>
						<td>
							<button type="button" class="btn btn-primary" v-on:click="addLoss">Add</button>
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

Vue.component('m-architecture-input-modal', {
	data: function() {
		return {
			componentIdx: '',
			layer: '',
			datasetIdx: '',
		};
	},
	props: ['components'],
	mounted: function() {
		$(this.$refs.modal).modal('show');
	},
	methods: {
		addDataset: function() {
			$(this.$refs.modal).modal('hide');
			this.$emit('success', {
				Type: 'dataset',
				DatasetIdx: parseInt(this.datasetIdx),
			});
		},
		addLayer: function() {
			$(this.$refs.modal).modal('hide');
			this.$emit('success', {
				Type: 'layer',
				ComponentIdx: parseInt(this.componentIdx),
				Layer: this.layer,
			});
		},
	},
	template: `
<div class="modal" tabindex="-1" role="dialog" ref="modal">
	<div class="modal-dialog modal-xl" role="document">
		<div class="modal-content">
			<div class="modal-body">
				<ul class="nav nav-tabs">
					<li class="nav-item">
						<a class="nav-link active" data-toggle="tab" href="#m-aim-dataset-tab" role="tab">Dataset</a>
					</li>
					<li class="nav-item">
						<a class="nav-link" data-toggle="tab" href="#m-aim-layer-tab" role="tab">Layer</a>
					</li>
				</ul>
				<div class="tab-content">
					<div class="tab-pane show active" id="m-aim-dataset-tab">
						<form v-on:submit.prevent="addDataset">
							<div class="form-group row">
								<label class="col-sm-2 col-form-label">Dataset Index</label>
								<div class="col-sm-10">
									<input class="form-control" type="text" v-model="datasetIdx" />
								</div>
							</div>
							<div class="form-group row">
								<div class="col-sm-10">
									<button type="submit" class="btn btn-primary">Add Dataset</button>
								</div>
							</div>
						</form>
					</div>
					<div class="tab-pane" id="m-aim-layer-tab">
						<form v-on:submit.prevent="addLayer">
							<div class="form-group row">
								<label class="col-sm-2 col-form-label">Component</label>
								<div class="col-sm-10">
									<select v-model="componentIdx" class="form-control">
										<template v-for="(comp, compIdx) in components">
											<option :key="compIdx" :value="compIdx">Component #{{ compIdx }}: {{ comp.Name }}</option>
										</template>
									</select>
								</div>
							</div>
							<div class="form-group row">
								<label class="col-sm-2 col-form-label">Layer</label>
								<div class="col-sm-10">
									<template v-if="componentIdx !== ''">
										<select v-model="layer" class="form-control">
											<template v-for="x in components[componentIdx].Params.Layers">
												<option :key="x" :value="x">{{ x }}</option>
											</template>
										</select>
									</template>
								</div>
							</div>
							<div class="form-group row">
								<div class="col-sm-10">
									<button type="submit" class="btn btn-primary">Add Layer</button>
								</div>
							</div>
						</form>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
	`,
});
