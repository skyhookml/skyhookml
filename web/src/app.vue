<template>
<div class="flex-container">

<header class="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow">
	<router-link class="navbar-brand col-md-3 col-lg-2 me-0 px-3" href="#" :to="wsPrefix">SkyhookML</router-link>
	<button class="navbar-toggler position-absolute d-md-none collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#sidebarMenu" aria-controls="sidebarMenu" aria-expanded="false" aria-label="Toggle navigation">
		<span class="navbar-toggler-icon"></span>
	</button>
	<div class="d-flex">
		<form class="d-flex align-items-center">
			<label class="mx-2">Workspace:</label>
			<select v-model="selectedWorkspace" @change="changedWorkspace" class="form-select form-select-sm mx-2">
				<option v-for="ws in workspaces" :key="ws" :value="ws">{{ ws }}</option>
			</select>
			<button type="button" class="btn btn-sm btn-danger mx-2" v-on:click="deleteWorkspace">Remove</button>
		</form>
		<form v-on:submit.prevent="createWorkspace" class="d-flex align-items-center ms-4">
			<input v-model="addForms.workspace.name" type="form-control form-control-sm" placeholder="New Workspace Name" class="mx-2" />
			<button type="submit" class="btn btn-sm btn-primary mx-2">New Workspace</button>
			<button type="button" class="btn btn-sm btn-primary mx-2" v-on:click="cloneWorkspace">Clone</button>
		</form>
	</div>
</header>


<div class="container-fluid flex-content">
<div class="row el-high">

	<nav id="sidebarMenu" class="col-md-3 col-lg-2 d-md-block bg-light sidebar collapse">
		<div class="position-sticky pt-3">
			<ul class="nav flex-column">
				<li class="nav-item">
					<router-link class="nav-link" :to="wsPrefix" active-class="active" exact>
						<i class="bi bi-speedometer"></i>&nbsp;
						Dashboard
					</router-link>
				</li>
				<!-- Show current quickstart page in sidebar. -->
				<template v-if="$route.path.includes('/quickstart/import')">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/quickstart/import'" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Quickstart: Import
						</router-link>
					</li>
				</template>
				<template v-if="$route.path.includes('/quickstart/train')">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/quickstart/train'" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Quickstart: Train
						</router-link>
					</li>
				</template>
				<template v-if="$route.path.includes('/quickstart/apply')">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/quickstart/apply'" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Quickstart: Apply
						</router-link>
					</li>
				</template>

				<li class="nav-item">
					<router-link class="nav-link" :to="wsPrefix + '/datasets'" active-class="active" exact>
						<i class="bi bi-files"></i>&nbsp;
						Datasets
					</router-link>
				</li>
				<!-- If viewing a dataset, show it in sidebar. -->
				<template v-if="$route.params.dsid && $store.state.routeData.dataset">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/datasets/'+$route.params.dsid" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Dataset: {{ $store.state.routeData.dataset.Name }}
						</router-link>
					</li>
				</template>
				<!-- If viewing an item, show it in sidebar. -->
				<template v-if="$route.params.itemkey">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/datasets/'+$route.params.dsid+'/items/'+$route.params.itemkey" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							<i class="bi bi-arrow-right"></i>&nbsp;
							Item: {{ $route.params.itemkey }}
						</router-link>
					</li>
				</template>

				<li class="nav-item">
					<router-link class="nav-link" :to="wsPrefix + '/annotate'" active-class="active" exact>
						<i class="bi bi-pencil-square"></i>&nbsp;
						Annotate
					</router-link>
				</li>
				<!-- If viewing an annoset, show it in sidebar. -->
				<template v-if="$route.params.setid && $store.state.routeData.annoset">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/annotate/'+$store.state.routeData.annoset.Tool+'/'+$route.params.setid" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Annotating: {{ $store.state.routeData.annoset.Dataset.Name }}
						</router-link>
					</li>
				</template>
				<!-- Show add annotation set in sidebar. -->
				<template v-if="$route.path.includes('/annotate-add')">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/annotate-add'" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Add Annotation Dataset
						</router-link>
					</li>
				</template>

				<li class="nav-item">
					<router-link class="nav-link" :to="wsPrefix + '/pipeline'" active-class="active" exact>
						<i class="bi bi-diagram-3"></i>&nbsp;
						Pipeline
					</router-link>
				</li>
				<!-- If editing a node, show it in sidebar. -->
				<template v-if="$route.params.nodeid && $route.path.includes('/exec/') && $store.state.routeData.node">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/exec/'+$route.params.nodeid" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Editing: {{ $store.state.routeData.node.Name }}
						</router-link>
					</li>
				</template>

				<li class="nav-item">
					<router-link class="nav-link" :to="wsPrefix + '/jobs'" active-class="active" exact>
						<i class="bi bi-list-task"></i>&nbsp;
						Jobs
					</router-link>
				</li>
				<!-- If viewing a job, show it in sidebar. -->
				<template v-if="$route.params.jobid && $store.state.routeData.job">
					<li class="nav-item">
						<router-link class="nav-link" :to="wsPrefix+'/jobs/'+$route.params.jobid" active-class="active" exact>
							<i class="bi bi-arrow-right"></i>&nbsp;
							Job: {{ $store.state.routeData.job.Name }}
						</router-link>
					</li>
				</template>

				<hr />
				<li class="nav-item">
					<router-link class="nav-link" :to="wsPrefix + '/models'" active-class="active" exact>
						<i class="bi bi-box-seam"></i>&nbsp;
						Model Architectures
					</router-link>
				</li>
			</ul>
		</div>
	</nav>

	<main class="col-md-9 ms-sm-auto col-lg-10 px-md-4 pt-3 pb-2 flex-container">
		<div v-if="error != ''" class="alert alert-danger alert-dismissible" role="alert">
			<strong>Error:</strong>
			{{ error }}
			<button type="button" class="btn-close" v-on:click="setError('')">
			</button>
		</div>
		<div class="flex-content">
			<router-view></router-view>
		</div>
	</main>
</div>
</div>

</div>
</template>

<script>
import utils from './utils.js';

import Dashboard from './dashboard.vue';

import QuickstartImport from './quickstart-import.vue';
import QuickstartTrain from './quickstart-train.vue';
import QuickstartApply from './quickstart-apply.vue';

import Datasets from './datasets.vue';
import Dataset from './dataset.vue';
import RenderItem from './render-item.vue';

import Annotate from './annotate.vue';
import AnnotateAdd from './annotate-add.vue';
import AnnotateInt from './annotate-int.js';
import AnnotateShape from './annotate-shape.js';
import AnnotateGeoJSON from './annotate-geojson.vue';
import AnnotateDetectionToTrack from './annotate-detection-to-track.js';

import Models from './models.vue';
import MArch from './m-architecture.vue';
import MComp from './m-component.vue';

import Pipeline from './pipeline.vue';
import ExecEdit from './exec-edit.vue';
import Compare from './compare.vue';
import Interactive from './interactive.vue';

import Jobs from './jobs.vue';
import Job from './job.vue';

const router = new VueRouter({
	routes: [
		{path: '/', redirect: '/ws/default'},

		{path: '/ws/:ws', component: Dashboard},

		{path: '/ws/:ws/quickstart/import', component: QuickstartImport},
		{path: '/ws/:ws/quickstart/train', component: QuickstartTrain},
		{path: '/ws/:ws/quickstart/apply', component: QuickstartApply},

		{path: '/ws/:ws/datasets', component: Datasets},
		{path: '/ws/:ws/datasets/:dsid', component: Dataset},
		{path: '/ws/:ws/datasets/:dsid/items/:itemkey', component: RenderItem},

		{path: '/ws/:ws/annotate', component: Annotate},
		{path: '/ws/:ws/annotate-add', component: AnnotateAdd},
		{path: '/ws/:ws/annotate/int/:setid', component: AnnotateInt},
		{path: '/ws/:ws/annotate/shape/:setid', component: AnnotateShape},
		{path: '/ws/:ws/annotate/detection-to-track/:setid', component: AnnotateDetectionToTrack},
		{path: '/ws/:ws/annotate/geojson/:setid', component: AnnotateGeoJSON},

		{path: '/ws/:ws/models', component: Models},
		{path: '/ws/:ws/models/arch/:archid', component: MArch},
		{path: '/ws/:ws/models/comp/:compid', component: MComp},

		{path: '/ws/:ws/pipeline', component: Pipeline},
		{path: '/ws/:ws/exec/:nodeid', component: ExecEdit},
		{path: '/ws/:ws/compare/:nodeid/:otherws/:othernodeid', component: Compare},
		{path: '/ws/:ws/interactive/:nodeid', component: Interactive},

		{path: '/ws/:ws/jobs', component: Jobs},
		{path: '/ws/:ws/jobs/:jobid', component: Job},
	],
});

// We use a simple store to keep track of data about the current route.
// For example, if we are viewing an item in a dataset, the object will have
// a reference to the dataset, and a reference to the item.
// This way, we can use that information both in the router view component, and
// in the sidebar.
const store = new Vuex.Store({
	state: {
		routeData: {},
	},
	mutations: {
		setRouteData(state, newData) {
			state.routeData = newData;
		},
	},
});

export default {
	router: router,
	store: store,
	data: {
		error: '',
		selectedWorkspace: '',
		workspaces: [],
		addForms: null,
	},
	created: function() {
		this.fetch();
		this.resetForm();

		if(this.$route.params.ws) {
			this.selectedWorkspace = this.$route.params.ws;
		}
	},
	methods: {
		fetch: function() {
			utils.request(this, 'GET', '/workspaces', null, (data) => {
				this.workspaces = data;
			});
		},
		resetForm: function() {
			this.addForms = {
				workspace: {
					name: '',
				},
			};
		},
		setPage: function(name) {
			if(!this.$route.params.ws) {
				return;
			}
			this.$router.push('/ws/'+this.$route.params.ws+'/'+name);
			this.setError('');
		},
		changedWorkspace: function() {
			this.$router.push('/ws/'+this.selectedWorkspace);
			this.resetForm();
		},
		createWorkspace: function() {
			let name = this.addForms.workspace.name;
			utils.request(this, 'POST', '/workspaces', {name: name}, () => {
				this.resetForm();
				this.fetch();
				this.selectedWorkspace = name;
				this.$router.push('/ws/'+name);
			});
		},
		cloneWorkspace: function() {
			let url = '/workspaces/'+this.$route.params.ws+'/clone';
			var params = {
				name: this.addForms.workspace.name,
			};
			utils.request(this, 'POST', url, params, () => {
				this.resetForm();
				this.fetch();
				this.$router.push('/ws/'+params.name);
			});
		},
		deleteWorkspace: function() {
			utils.request(this, 'DELETE', '/workspaces/'+this.selectedWorkspace, null, () => {
				this.fetch();
				this.$router.push('/');
			});
		},
		setError: function(error) {
			this.error = error;
		},
	},
	computed: {
		wsPrefix: function() {
			if(this.$route.params.ws) {
				return '/ws/' + this.$route.params.ws;
			} else if(this.selectedWorkspace) {
				return '/ws' + this.selectedWorkspace;
			} else if(this.workspaces.length > 0) {
				return '/ws/' + this.workspaces[0];
			} else {
				return '/';
			}
		},
	},
	watch: {
		$route: function(to, from) {
			this.$store.commit('setRouteData', {});
		},
	},
};
</script>
