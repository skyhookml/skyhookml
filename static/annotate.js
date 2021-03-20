import utils from './utils.js';

const Annotate = {
	data: function() {
		return {
			annosets: [],
			datasets: [],
			addForm: {},

			tools: [
				{ID: "shape", Name: "Shapes and Detections"},
				{ID: "int", Name: "Integers (e.g., classes, categories)"},
				{ID: "detection-to-track", Name: "Group Detections into Tracks"},
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
		resetAddForm: function() {
			this.addForm = {
				dataset: '',
				inputs: [],
				tool: '',
				inputSelection: '',
			};
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
			var inputIDs = this.addForm.inputs.map((ds) => ds.ID);
			let params = {
				ds_id: this.addForm.dataset,
				inputs: inputIDs.join(','),
				tool: this.addForm.tool,
				params: '',
			};
			utils.request(this, 'POST', '/annotate-datasets', params, () => {
				$(this.$refs.addModal).modal('hide');
				this.fetch(true);
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
							<label class="col-sm-2 col-form-label">Dataset</label>
							<div class="col-sm-10">
								<select v-model="addForm.dataset" class="form-select">
									<option v-for="ds in datasets" :key="ds.ID" :value="ds.ID">{{ ds.Name }}</option>
								</select>
							</div>
						</div>
						<div class="row mb-2">
							<label class="col-sm-2 col-form-label">Inputs</label>
							<div class="col-sm-10">
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
						</div>
						<div class="row mb-2">
							<label class="col-sm-2 col-form-label">Tool</label>
							<div class="col-sm-10">
								<select v-model="addForm.tool" class="form-select">
									<option v-for="toolObj in tools" :value="toolObj.ID">{{ toolObj.Name }}</option>
								</select>
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
				<th>Data Type</th>
				<th></th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="set in annosets">
				<td>{{ set.Dataset.Name }}</td>
				<td>{{ set.Tool }}</td>
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
