import utils from './utils.js';
import ArchInputModal from './m-architecture-input-modal.js';

export default {
	components: {
		'm-architecture-input-modal': ArchInputModal,
	},
	data: function() {
		return {
			arch: null,
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
	created: function() {
		this.resetForm();

		utils.request(this, 'GET', '/pytorch/components', null, (comps) => {
			comps.forEach((comp) => {
				this.$set(this.comps, comp.ID, comp);
			});
		});

		const archID = this.$route.params.archid;
		utils.request(this, 'GET', '/pytorch/archs/'+archID, null, (arch) => {
			this.arch = arch;
			const params = this.arch.Params;
			this.numInputs = params.NumInputs;
			this.numTargets = params.NumTargets;
			if(params.Components) {
				this.components = params.Components;
			}
			if(params.Losses) {
				this.losses = params.Losses;
			}
		});
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
			utils.request(this, 'POST', '/pytorch/archs/'+this.arch.ID, JSON.stringify({
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
	<template v-if="arch != null">
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
	</template>
</div>
	`,
};
