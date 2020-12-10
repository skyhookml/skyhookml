export default {
	data: function() {
		return {
			componentIdx: '',
			layer: '',
			datasetIdx: '',
		};
	},
	props: ['components'],
	mounted: function() {
		$(this.$refs.modal).modal('show');
	},
	methods: {
		addDataset: function() {
			$(this.$refs.modal).modal('hide');
			this.$emit('success', {
				Type: 'dataset',
				DatasetIdx: parseInt(this.datasetIdx),
			});
		},
		addLayer: function() {
			$(this.$refs.modal).modal('hide');
			this.$emit('success', {
				Type: 'layer',
				ComponentIdx: parseInt(this.componentIdx),
				Layer: this.layer,
			});
		},
	},
	template: `
<div class="modal" tabindex="-1" role="dialog" ref="modal">
	<div class="modal-dialog modal-xl" role="document">
		<div class="modal-content">
			<div class="modal-body">
				<ul class="nav nav-tabs">
					<li class="nav-item">
						<a class="nav-link active" data-toggle="tab" href="#m-aim-dataset-tab" role="tab">Dataset</a>
					</li>
					<li class="nav-item">
						<a class="nav-link" data-toggle="tab" href="#m-aim-layer-tab" role="tab">Layer</a>
					</li>
				</ul>
				<div class="tab-content">
					<div class="tab-pane show active" id="m-aim-dataset-tab">
						<form v-on:submit.prevent="addDataset">
							<div class="form-group row">
								<label class="col-sm-2 col-form-label">Dataset Index</label>
								<div class="col-sm-10">
									<input class="form-control" type="text" v-model="datasetIdx" />
								</div>
							</div>
							<div class="form-group row">
								<div class="col-sm-10">
									<button type="submit" class="btn btn-primary">Add Dataset</button>
								</div>
							</div>
						</form>
					</div>
					<div class="tab-pane" id="m-aim-layer-tab">
						<form v-on:submit.prevent="addLayer">
							<div class="form-group row">
								<label class="col-sm-2 col-form-label">Component</label>
								<div class="col-sm-10">
									<select v-model="componentIdx" class="form-control">
										<template v-for="(comp, compIdx) in components">
											<option :key="compIdx" :value="compIdx">Component #{{ compIdx }}: {{ comp.Name }}</option>
										</template>
									</select>
								</div>
							</div>
							<div class="form-group row">
								<label class="col-sm-2 col-form-label">Layer</label>
								<div class="col-sm-10">
									<template v-if="componentIdx !== ''">
										<select v-model="layer" class="form-control">
											<template v-for="x in components[componentIdx].Params.Layers">
												<option :key="x" :value="x">{{ x }}</option>
											</template>
										</select>
									</template>
								</div>
							</div>
							<div class="form-group row">
								<div class="col-sm-10">
									<button type="submit" class="btn btn-primary">Add Layer</button>
								</div>
							</div>
						</form>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
	`,
};
