Vue.component('annotate-tab', {
	data: function() {
		return {
			annosets: [],
			datasets: [],
			addForm: {},
			selectedAnnoset: null,
		};
	},
	props: ['tab'],
	created: function() {
		this.resetAddForm();
		this.fetch(true);
		//setInterval(this.fetch, 1000);
	},
	methods: {
		fetch: function(force) {
			if(!force && this.tab != '#annotate-panel') {
				return;
			}
			myCall('GET', '/annotate-datasets', null, (data) => {
				this.annosets = data;
			});
			myCall('GET', '/datasets', null, (data) => {
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
		add: function() {
			var inputIDs = this.addForm.inputs.map((ds) => ds.ID);
			let params = {
				ds_id: this.addForm.dataset,
				inputs: inputIDs.join(','),
				tool: this.addForm.tool,
				params: '',
			};
			myCall('POST', '/annotate-datasets', params, () => {
				$(this.$refs.addModal).modal('hide');
				this.fetch(true);
			});
		},
		select: function(annoset) {
			this.selectedAnnoset = annoset;
		},
	},
	watch: {
		tab: function() {
			if(this.tab != '#annotate-panel') {
				return;
			}
			this.fetch(true);
		},
	},
	template: `
<div>
	<template v-if="selectedAnnoset == null">
		<div class="my-1">
			<button type="button" class="btn btn-primary" v-on:click="showAddModal">Add Annotation Dataset</button>
			<div class="modal" tabindex="-1" role="dialog" ref="addModal">
				<div class="modal-dialog" role="document">
					<div class="modal-content">
						<div class="modal-body">
							<form v-on:submit.prevent="add">
								<div class="form-group row">
									<label class="col-sm-4 col-form-label">Dataset</label>
									<div class="col-sm-8">
										<select v-model="addForm.dataset" class="form-control">
											<option v-for="ds in datasets" :key="ds.ID" :value="ds.ID">{{ ds.Name }}</option>
										</select>
									</div>
								</div>
								<div class="form-group row">
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
														<select v-model="addForm.inputSelection" class="form-control">
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
								<div class="form-group row">
									<label class="col-sm-4 col-form-label">Tool</label>
									<div class="col-sm-8">
										<input class="form-control" type="text" v-model="addForm.tool" />
									</div>
								</div>
								<div class="form-group row">
									<div class="col-sm-8">
										<button type="submit" class="btn btn-primary">Add Annotate Dataset</button>
									</div>
								</div>
							</form>
						</div>
					</div>
				</div>
			</div>
		</div>
		<table class="table">
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
						<button v-on:click="select(set)" class="btn btn-primary">Annotate</button>
						<!--<button v-on:click="deleteDataset(set.ID)" class="btn btn-danger">Delete</button>-->
					</td>
				</tr>
			</tbody>
		</table>
	</template>
	<template v-else>
		<h2>
			<a href="#" v-on:click.prevent="select(null)">Annotate</a>
			/
			{{ selectedAnnoset.Dataset.Name }}
		</h2>
		<component v-bind:is="'annotate-'+selectedAnnoset.Tool" v-bind:annoset="selectedAnnoset"></component>
	</template>
</div>
	`,
});
