<template>
<div>
	<button type="button" class="btn btn-primary my-1" v-on:click="showAddModal">Add Architecture</button>
	<div class="modal" tabindex="-1" role="dialog" ref="addModal">
		<div class="modal-dialog" role="document">
			<div class="modal-content">
				<div class="modal-body">
					<form v-on:submit.prevent="add">
						<div class="row mb-2">
							<label class="col-sm-4 col-form-label">ID</label>
							<div class="col-sm-8">
								<input class="form-control" type="text" v-model="addForm.id" required />
							</div>
						</div>
						<div class="row">
							<div class="col-sm-8">
								<button type="submit" class="btn btn-primary">Add Architecture</button>
							</div>
						</div>
					</form>
				</div>
			</div>
		</div>
	</div>
	<table class="table table-sm">
		<thead>
			<tr>
				<th>ID</th>
				<th></th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="arch in archs">
				<td>{{ arch.ID }}</td>
				<td>
					<router-link :to="'/ws/'+$route.params.ws+'/models/arch/'+arch.ID" class="btn btn-sm btn-primary">Manage</router-link>
					<button v-on:click="deleteArch(arch.ID)" class="btn btn-sm btn-danger">Delete</button>
				</td>
			</tr>
		</tbody>
	</table>
</div>
</template>

<script>
import utils from './utils.js';

export default {
	data: function() {
		return {
			archs: [],
			addForm: {},
		};
	},
	props: ['mtab'],
	created: function() {
		this.fetch();
	},
	methods: {
		fetch: function() {
			utils.request(this, 'GET', '/pytorch/archs', null, (data) => {
				this.archs = data;
			});
		},
		showAddModal: function() {
			this.addForm = {
				id: '',
			};
			$(this.$refs.addModal).modal('show');
		},
		add: function() {
			utils.request(this, 'POST', '/pytorch/archs', this.addForm, () => {
				$(this.$refs.addModal).modal('hide');
				this.fetch();
			});
		},
		deleteArch: function(archID) {
			utils.request(this, 'DELETE', '/pytorch/archs/'+archID, null, () => {
				this.fetch();
			});
		},
	},
	watch: {
		tab: function() {
			if(this.mtab != '#m-architectures-panel') {
				return;
			}
			this.fetch();
		},
	},
};
</script>