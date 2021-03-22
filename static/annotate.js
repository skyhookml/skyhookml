import utils from './utils.js';

const Annotate = {
	data: function() {
		return {
			annosets: [],
			datasets: [],
			addForm: {},

			tools: [
				{
					ID: "shape",
					Name: "Shapes and Detections",
					Inputs: [{
						Name: "Image/Video",
						DataTypes: ["image", "video"],
					}],
					DataTypes: ["shape", "detection"],
				},
				{
					ID: "int",
					Name: "Integers (e.g., classes, categories)",
					Inputs: [{
						Name: "Image/Video",
						DataTypes: ["image", "video"],
					}],
					DataTypes: ["int"],
				},
				{
					ID: "detection-to-track",
					Name: "Group Detections into Tracks",
					Inputs: [{
						Name: "Image/Video",
						DataTypes: ["image", "video"],
					}, {
						Name: "Detections",
						DataTypes: ["detection"],
					}],
					DataTypes: ["detection"],
				},
			],
		};
	},
	created: function() {
		this.resetAddForm();
		this.fetch();
	},
	methods: {
		fetch: function() {
			utils.request(this, 'GET', '/annotate-datasets', null, (data) => {
				this.annosets = data;
			});
			utils.request(this, 'GET', '/datasets', null, (data) => {
				this.datasets = data;
			});
		},
		selectAnnoset: function(annoset) {
			this.$router.push('/ws/'+this.$route.params.ws+'/annotate/'+annoset.Tool+'/'+annoset.ID);
		},
		removeAnnoset: function(annoset) {
			utils.request(this, 'DELETE', '/annotate-datasets/'+annoset.ID, null, () => {
				this.fetch();
			});
		},

		// Functions for add form.
		resetAddForm: function() {
			this.addForm = {
				// the selected tool ID and actual object
				// object is filled in by changedTool
				tool: '',
				toolObj: null,

				// either new or existing, depending on whether a new dataset should be created
				datasetMode: '',

				// name and type of dataset in case datasetMode=='new'
				datasetName: '',
				datasetType: '',

				// ID of existing dataset in case datasetMode=='existing'
				datasetID: '',

				// input datasets to use for annotation
				// if toolObj.Inputs is set, we use inputIDs and inputs must correspond to that configuration
				// otherwise, we use inputs which is a list of dataset objects
				inputs: [],
				inputIDs: [],

				// the currently selected input dataset (only used if toolObj.Inputs is not set)
				inputSelection: '',
			};
		},
		changedTool: function() {
			// update the cached addForm.toolObj
			// if the newly selected tool has Inputs set, then we also initialize
			//   null entries in addForm.inputs corresponding to each toolObj.Inputs
			// similarly, if it has a single DataTypes, we set datasetType
			this.addForm.toolObj = null;
			this.addForm.inputs = [];
			this.addForm.inputIDs = [];
			this.addForm.datasetType = '';
			this.tools.forEach((toolObj) => {
				if(toolObj.ID == this.addForm.tool) {
					this.addForm.toolObj = toolObj;
				}
			});
			let toolObj = this.addForm.toolObj;
			if(toolObj && toolObj.Inputs) {
				for(let i = 0; i < toolObj.Inputs.length; i++) {
					this.addForm.inputIDs.push(null);
				}
			}
			if(toolObj && toolObj.DataTypes && toolObj.DataTypes.length == 1) {
				this.addForm.datasetType = toolObj.DataTypes[0];
			}
		},
		showAddModal: function() {
			this.resetAddForm();
			$(this.$refs.addModal).modal('show');
		},
		formAddInput: function(ds) {
			let dsID = parseInt(this.addForm.inputSelection);
			let dataset = this.datasets.filter((ds) => ds.ID == dsID)[0];
			this.addForm.inputs.push(dataset);
			this.addForm.inputSelection = '';
		},
		formRemoveInput: function(i) {
			this.addForm.inputs.splice(i, 1);
		},
		addAnnoset: function() {
			let handle = async () => {
				let datasetID = null;
				if(this.addForm.datasetMode == 'new') {
					let params = {
						name: this.addForm.datasetName,
						data_type: this.addForm.datasetType,
					};
					await utils.request(this, 'POST', '/datasets', params, (ds) => {
						datasetID = ds.ID;
					});
				} else if(this.addForm.datasetMode == 'existing') {
					datasetID = this.addForm.datasetID;
				}

				let inputIDs = null;
				if(this.addForm.inputIDs.length > 0) {
					inputIDs = this.addForm.inputIDs;
				} else {
					inputIDs = this.addForm.inputs.map((ds) => ds.ID);
				}

				let params = {
					ds_id: datasetID,
					inputs: inputIDs.join(','),
					tool: this.addForm.tool,
					params: '',
				};
				await utils.request(this, 'POST', '/annotate-datasets', params, () => {
					$(this.$refs.addModal).modal('hide');
					this.fetch(true);
				});
			};
			handle();
		},
	},
	filters: {
		// Format the Inputs of an annotation dataset.
		niceInputs: function(inputs) {
			let datasetNames = inputs.map((input) => input.Name);
			return datasetNames.join(', ');
		},
	},
	template: `
<div>
	<div class="border-bottom mb-3">
		<h2>Annotate</h2>
	</div>
	<button type="button" class="btn btn-primary mb-2" v-on:click="showAddModal">Add Annotation Dataset</button>
	<div class="modal" tabindex="-1" role="dialog" ref="addModal">
		<div class="modal-dialog" role="document">
			<div class="modal-content">
				<div class="modal-body">
					<form v-on:submit.prevent="addAnnoset">
						<div class="row mb-2">
							<label class="col-sm-2 col-form-label">Tool</label>
							<div class="col-sm-10">
								<select v-model="addForm.tool" class="form-select" @change="changedTool">
									<option v-for="toolObj in tools" :value="toolObj.ID">{{ toolObj.Name }}</option>
								</select>
							</div>
						</div>
						<div class="row mb-2">
							<label class="col-sm-2 col-form-label">Dataset</label>
							<div class="col-sm-10">
								<div class="form-check">
									<input class="form-check-input" type="radio" v-model="addForm.datasetMode" value="new">
									<label class="form-check-label">Create a new dataset</label>
								</div>
								<div class="form-check">
									<input class="form-check-input" type="radio" v-model="addForm.datasetMode" value="existing">
									<label class="form-check-label">Use an existing dataset</label>
								</div>
							</div>
						</div>
						<template v-if="addForm.datasetMode == 'new'">
							<div class="row mb-2">
								<label class="col-sm-2 col-form-label">Name</label>
								<div class="col-sm-10">
									<input v-model="addForm.datasetName" type="text" class="form-control">
								</div>
							</div>
							<div class="row mb-2" v-if="addForm.toolObj">
								<label class="col-sm-2 col-form-label">Data Type</label>
								<div class="col-sm-10">
									<template v-if="!addForm.toolObj.DataTypes || addForm.toolObj.DataTypes.length != 1">
										<select v-model="addForm.datasetType" class="form-select">
											<template v-for="(name, dt) in $globals.dataTypes">
												<option
													v-if="!addForm.toolObj.DataTypes || addForm.toolObj.DataTypes.includes(dt)"
													:value="dt"
													>
													{{ name }}
												</option>
											</template>
										</select>
									</template>
									<template v-else>
										<input type="text" readonly class="form-control-plaintext" :value="addForm.datasetType" />
									</template>
								</div>
							</div>
						</template>
						<template v-if="addForm.datasetMode == 'existing' && addForm.toolObj">
							<div class="row mb-2" >
								<div class="col-sm-10 offset-sm-2">
									<select v-model="addForm.datasetID" class="form-select">
										<template v-for="ds in datasets">
											<!-- Only show datasets that match the type of the selected tool. -->
											<option
												v-if="ds.Type != 'computed' && (!addForm.toolObj.DataTypes || addForm.toolObj.DataTypes.includes(ds.DataType))"
												:key="ds.ID"
												:value="ds.ID">
												{{ ds.Name }}
											</option>
										</template>
									</select>
								</div>
							</div>
						</template>
						<div class="row mb-2">
							<label class="col-sm-2 col-form-label">Inputs</label>
							<div class="col-sm-10" v-if="!addForm.toolObj || !addForm.toolObj.Inputs">
								<table class="table">
									<tbody>
										<tr v-for="(ds, i) in addForm.inputs">
											<td>{{ ds.Name }}</td>
											<td>
												<button type="button" class="btn btn-danger" v-on:click="formRemoveInput(i)">Remove</button>
											</td>
										</tr>
										<tr>
											<td>
												<select v-model="addForm.inputSelection" class="form-select">
													<option v-for="ds in datasets" :key="ds.ID" :value="ds.ID">{{ ds.Name }}</option>
												</select>
											</td>
											<td>
												<button type="button" class="btn btn-primary" v-on:click="formAddInput">Add</button>
											</td>
										</tr>
									</tbody>
								</table>
							</div>
							<div class="col-sm-10" v-else>
								<table class="table">
									<thead>
										<tr>
											<th>Input Name</th>
											<th>Dataset</th>
										</tr>
									</thead>
									<tbody>
										<tr v-for="(input, i) in addForm.toolObj.Inputs">
											<td>{{ input.Name }}</td>
											<td>
												<select v-model="addForm.inputIDs[i]" class="form-select">
													<template v-for="ds in datasets">
														<!-- Only show datasets that match the type of this input. -->
														<option
															v-if="!input.DataTypes || input.DataTypes.includes(ds.DataType)"
															:key="ds.ID"
															:value="ds.ID">
															{{ ds.Name }}
														</option>
													</template>
												</select>
											</td>
										</tr>
									</tbody>
								</table>
							</div>
						</div>
						<div class="row mb-2">
							<div class="col-sm-10">
								<button type="submit" class="btn btn-primary">Add Annotate Dataset</button>
							</div>
						</div>
					</form>
				</div>
			</div>
		</div>
	</div>
	<table class="table table-sm align-middle">
		<thead>
			<tr>
				<th>Name</th>
				<th>Tool</th>
				<th>Inputs</th>
				<th>Data Type</th>
				<th></th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="set in annosets">
				<td>{{ set.Dataset.Name }}</td>
				<td>{{ set.Tool }}</td>
				<td>{{ set.Inputs | niceInputs }}</td>
				<td>{{ set.Dataset.DataType }}</td>
				<td>
					<button v-on:click="selectAnnoset(set)" class="btn btn-sm btn-primary">Annotate</button>
					<button v-on:click="removeAnnoset(set)" class="btn btn-sm btn-danger">Remove</button>
				</td>
			</tr>
		</tbody>
	</table>
</div>
	`,
};
export default Annotate;
