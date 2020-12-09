Vue.component('datasets-tab', {
	data: function() {
		return {
			datasets: [],
			addDatasetForm: {},
			selectedDataset: null,
		};
	},
	props: ['tab'],
	created: function() {
		this.fetchDatasets(true);
		//setInterval(this.fetchDatasets, 1000);
	},
	methods: {
		fetchDatasets: function(force) {
			if(!force && this.tab != '#datasets-panel') {
				return;
			}
			myCall('GET', '/datasets', null, (data) => {
				this.datasets = data;
			});
		},
		showAddDatasetModal: function() {
			this.addDatasetForm = {
				name: '',
				data_type: '',
			};
			$(this.$refs.addDatasetModal).modal('show');
		},
		addDataset: function() {
			myCall('POST', '/datasets', this.addDatasetForm, () => {
				$(this.$refs.addDatasetModal).modal('hide');
				this.fetchDatasets(true);
			});
		},
		deleteDataset: function(dsID) {
			myCall('DELETE', '/timelines/'+dsID, null, () => {
				this.fetchDatasets(true);
			});
		},
		selectDataset: function(dataset) {
			this.selectedDataset = dataset;
		},
	},
	watch: {
		tab: function() {
			if(this.tab != '#datasets-panel') {
				return;
			}
			this.fetchDatasets(true);
		},
	},
	template: `
<div>
	<template v-if="selectedDataset == null">
		<div class="my-1">
			<button type="button" class="btn btn-primary" v-on:click="showAddDatasetModal">Add Dataset</button>
			<div class="modal" tabindex="-1" role="dialog" ref="addDatasetModal">
				<div class="modal-dialog" role="document">
					<div class="modal-content">
						<div class="modal-body">
							<form v-on:submit.prevent="addDataset">
								<div class="form-group row">
									<label class="col-sm-4 col-form-label">Name</label>
									<div class="col-sm-8">
										<input class="form-control" type="text" v-model="addDatasetForm.name" />
									</div>
								</div>
								<div class="form-group row">
									<label class="col-sm-4 col-form-label">Data Type</label>
									<div class="col-sm-8">
										<select v-model="addDatasetForm.data_type" class="form-control">
											<option value="image">Image</option>
											<option value="video">Video</option>
											<option value="detection">Detection</option>
											<option value="track">Track</option>
											<option value="shape">Shape</option>
											<option value="int">Integer</option>
											<option value="floats">Floats</option>
											<option value="imlist">Image List</option>
											<option value="text">Text</option>
											<option value="string">String</option>
										</select>
									</div>
								</div>
								<div class="form-group row">
									<div class="col-sm-8">
										<button type="submit" class="btn btn-primary">Add Dataset</button>
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
					<th>Type</th>
					<th>Data Type</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="ds in datasets">
					<td>{{ ds.Name }}</td>
					<td>{{ ds.Type }}</td>
					<td>{{ ds.DataType }}</td>
					<td>
						<button v-on:click="selectDataset(ds)" class="btn btn-primary">Manage</button>
						<button v-on:click="deleteDataset(ds.ID)" class="btn btn-danger">Delete</button>
					</td>
				</tr>
			</tbody>
		</table>
	</template>
	<template v-else>
		<dataset v-bind:dataset="selectedDataset" v-on:back="selectDataset(null)"></dataset>
	</template>
</div>
	`,
});
