<template>
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
					<h3>
						Component #{{ compIdx }}: {{ compSpec.ID }}
						<button type="button" class="btn btn-sm btn-danger" v-on:click="removeComponent(compIdx)">Remove</button>
					</h3>
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
				<form class="row g-1 align-items-center" v-on:submit.prevent="addComponent">
					<div class="col-auto">
						<select class="form-select" v-model="addComponentForm.componentID">
							<option v-for="comp in comps" :key="comp.ID" :value="comp.ID">{{ comp.ID }}</option>
						</select>
					</div>
					<div class="col-auto">
						<button type="submit" class="btn btn-primary my-1 mx-1">Add Component</button>
					</div>
				</form>
			</div>
		</div>
		<template v-if="addInputModal">
			<m-architecture-input-modal v-bind:components="parentComponentList(addInputModal.componentIdx)" v-on:success="addInput($event)"></m-architecture-input-modal>
		</template>
		<template v-for="{label, list, form} in lossAndScore">
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">{{ label }}</label>
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
							<tr v-for="(spec, i) in list">
								<td>
									Component #{{ spec.ComponentIdx }}<template v-if="getComponent(spec.ComponentIdx)">: {{ getComponent(spec.ComponentIdx).ID }}</template>
								</td>
								<td>{{ spec.Layer }}</td>
								<td>{{ spec.Weight }}</td>
								<td>
									<button type="button" class="btn btn-danger" v-on:click="removeLoss(list, i)">Remove</button>
								</td>
							</tr>
							<tr>
								<td>
									<select v-model="form.componentIdx" class="form-select">
										<template v-for="(compSpec, compIdx) in components">
											<option v-if="compSpec.ID in comps" :key="compIdx" :value="compIdx">Component #{{ compIdx }}: {{ compSpec.ID }}</option>
										</template>
									</select>
								</td>
								<td>
									<template v-if="getComponent(form.componentIdx)">
										<select v-model="form.layer" class="form-select">
											<template v-for="layer in getComponent(form.componentIdx).Params.Losses">
												<option :key="layer" :value="layer">{{ layer }}</option>
											</template>
										</select>
									</template>
								</td>
								<td>
									<input class="form-control" type="text" v-model.number="form.weight" />
								</td>
								<td>
									<button type="button" class="btn btn-primary" v-on:click="addLoss(list, form)">Add</button>
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
import utils from './utils.js';
import ArchInputModal from './m-architecture-input-modal.vue';

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
			scores: [],

			addComponentForm: null,
			addLossForm: null,
			addScoreForm: null,

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
			if(params.Scores) {
				this.scores = params.Scores;
			}
		});
	},
	methods: {
		resetForm: function() {
			this.addComponentForm = {
				componentID: '',
			};
			this.addLossForm = {
				componentIdx: '',
				layer: '',
				weight: 1.0,
			};
			this.addScoreForm = {
				componentIdx: '',
				layer: '',
				weight: 1.0,
			};
		},
		save: function() {
			let params = {
				NumInputs: parseInt(this.numInputs),
				NumTargets: parseInt(this.numTargets),
				Components: this.components,
				Losses: this.losses,
				Scores: this.scores,
			};
			utils.request(this, 'POST', '/pytorch/archs/'+this.arch.ID, JSON.stringify({
				Params: params,
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/models');
			});
		},
		addComponent: function() {
			this.components.push({
				ID: this.addComponentForm.componentID,
				Params: '',
				Inputs: [],
				Targets: [],
			});
			this.resetForm();
		},
		removeComponent: function(i) {
			this.components.splice(i, 1);
		},
		addLoss: function(list, form) {
			list.push({
				ComponentIdx: parseInt(form.componentIdx),
				Layer: form.layer,
				Weight: form.weight,
			});
			this.resetForm();
		},
		removeLoss: function(list, i) {
			list.splice(i, 1);
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
	computed: {
		lossAndScore: function() {
			return [{
				label: 'Losses',
				list: this.losses,
				form: this.addLossForm,
			}, {
				label: 'Scores',
				list: this.scores,
				form: this.addScoreForm,
			}];
		},
	},
};
</script>
