Vue.component('m-architectures', {
	data: function() {
		return {
			archs: [],
			addForm: {},
			selectedArch: null,
		};
	},
	props: ['mtab'],
	created: function() {
		this.fetch(true);
		//setInterval(this.fetch, 1000);
	},
	methods: {
		fetch: function(force) {
			if(!force && this.mtab != '#m-architectures-panel') {
				return;
			}
			myCall('GET', '/keras/archs', null, (data) => {
				this.archs = data;
			});
		},
		showAddModal: function() {
			this.addForm = {
				name: '',
			};
			$(this.$refs.addModal).modal('show');
		},
		add: function() {
			myCall('POST', '/keras/archs', this.addForm, () => {
				$(this.$refs.addModal).modal('hide');
				this.fetch(true);
			});
		},
		deleteArch: function(archID) {
			myCall('DELETE', '/keras/archs/'+archID, null, () => {
				this.fetch(true);
			});
		},
		selectArch: function(arch) {
			this.selectedArch = arch;
		},
		back: function() {
			this.fetch(true);
			this.selectArch(null);
		},
	},
	watch: {
		tab: function() {
			if(this.mtab != '#m-architectures-panel') {
				return;
			}
			this.fetch(true);
		},
	},
	template: `
<div>
	<template v-if="selectedArch == null">
		<div class="my-1">
			<button type="button" class="btn btn-primary" v-on:click="showAddModal">Add Architecture</button>
			<div class="modal" tabindex="-1" role="dialog" ref="addModal">
				<div class="modal-dialog" role="document">
					<div class="modal-content">
						<div class="modal-body">
							<form v-on:submit.prevent="add">
								<div class="form-group row">
									<label class="col-sm-4 col-form-label">Name</label>
									<div class="col-sm-8">
										<input class="form-control" type="text" v-model="addForm.name" />
									</div>
								</div>
								<div class="form-group row">
									<div class="col-sm-8">
										<button type="submit" class="btn btn-primary">Add Architecture</button>
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
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="arch in archs">
					<td>{{ arch.Name }}</td>
					<td>
						<button v-on:click="selectArch(arch)" class="btn btn-primary">Manage</button>
						<button v-on:click="deleteArch(arch.ID)" class="btn btn-danger">Delete</button>
					</td>
				</tr>
			</tbody>
		</table>
	</template>
	<template v-else>
		<div>
			<button type="button" class="btn btn-primary" v-on:click="back">Back</button>
		</div>
		<m-architecture v-bind:arch="selectedArch"></m-architecture>
	</template>
</div>
	`,
});
