Vue.component('m-components', {
	data: function() {
		return {
			comps: [],
			addForm: {},
			selectedComp: null,
		};
	},
	props: ['mtab'],
	created: function() {
		this.fetch(true);
		//setInterval(this.fetch, 1000);
	},
	methods: {
		fetch: function(force) {
			if(!force && this.mtab != '#m-components-panel') {
				return;
			}
			myCall('GET', '/pytorch/components', null, (data) => {
				this.comps = data;
			});
		},
		showAddModal: function() {
			this.addForm = {
				name: '',
			};
			$(this.$refs.addModal).modal('show');
		},
		add: function() {
			myCall('POST', '/pytorch/components', this.addForm, () => {
				$(this.$refs.addModal).modal('hide');
				this.fetch(true);
			});
		},
		deleteComp: function(compID) {
			myCall('DELETE', '/pytorch/components/'+compID, null, () => {
				this.fetch(true);
			});
		},
		selectComp: function(comp) {
			this.selectedComp = comp;
		},
		back: function() {
			this.fetch(true);
			this.selectComp(null);
		},
	},
	watch: {
		tab: function() {
			if(this.mtab != '#m-components-panel') {
				return;
			}
			this.fetch(true);
		},
	},
	template: `
<div>
	<template v-if="selectedComp == null">
		<div class="my-1">
			<button type="button" class="btn btn-primary" v-on:click="showAddModal">Add Component</button>
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
										<button type="submit" class="btn btn-primary">Add Component</button>
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
				<tr v-for="comp in comps">
					<td>{{ comp.Name }}</td>
					<td>
						<button v-on:click="selectComp(comp)" class="btn btn-primary">Manage</button>
						<button v-on:click="deleteComp(comp.ID)" class="btn btn-danger">Delete</button>
					</td>
				</tr>
			</tbody>
		</table>
	</template>
	<template v-else>
		<div>
			<button type="button" class="btn btn-primary" v-on:click="back">Back</button>
		</div>
		<m-component v-bind:comp="selectedComp"></m-component>
	</template>
</div>
	`,
});
