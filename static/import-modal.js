import utils from './utils.js';

export default {
	data: function() {
		return {
			path: '',
			symlink: false,
			file: null,
			percent: null,

			// either /datasets/[ID]/import or /import-dataset depending on mode
			importURL: '',
		};
	},
	props: [
		// 'add' if we're adding to an existing dataset
		// 'new' if we're importing a Skyhook export as a new dataset
		'mode',

		// the dataset object, only if mode='add'
		'dataset',
	],
	created: function() {
		if(this.mode == 'add') {
			this.importURL = '/datasets/'+this.dataset.ID+'/import';
		} else if(this.mode == 'new') {
			this.importURL = '/import-dataset';
		}
	},
	mounted: function() {
		$(this.$refs.modal).on('shown.bs.modal', () => {
			$(this.$refs.modal).focus();
		});
	},
	methods: {
		click: function() {
			this.path = '';
			this.file = null;
			this.percent = null;
			$(this.$refs.modal).modal('show');
		},
		onFileChange: function(event) {
			this.file = event.target.files[0];
		},
		submitLocal: function() {
			let params = {
				mode: 'local',
				path: this.path,
				symlink: this.symlink,
			};
			utils.request(this, 'POST', this.importURL+'?mode=local', params);
			$(this.$refs.modal).modal('hide');
		},
		submitUpload: function() {
			var data = new FormData();
			data.append('file', this.file);
			this.percent = null;
			$.ajax({
				type: 'POST',
				url: this.importURL+'?mode=upload',
				error: (req, status, errorMsg) => {
					app.setError(errorMsg);
				},
				data: data,
				processData: false,
				contentType: false,
				xhr: () => {
					var xhr = new window.XMLHttpRequest();
					xhr.upload.addEventListener('progress', (e) => {
						if(!e.lengthComputable) {
							return;
						}
						this.percent = parseInt(e.loaded * 100 / e.total);
					});
					return xhr;
				},
				success: (job) => {
					$(this.$refs.modal).modal('hide');
					this.$emit('imported', job);
					this.$router.push('/ws/'+this.$route.params.ws+'/jobs/'+job.ID);
				},
			});
		},
	},
	template: `
<span>
	<button type="button" class="btn btn-primary" v-on:click="click">
		<template v-if="mode == 'add'">
			Import Data
		</template>
		<template v-else-if="mode == 'new'">
			Import SkyhookML Dataset
		</template>
	</button>
	<div class="modal" tabindex="-1" role="dialog" ref="modal">
		<div class="modal-dialog" role="document">
			<div class="modal-content">
				<div class="modal-body">
					<ul class="nav nav-tabs mb-2">
						<li class="nav-item">
							<a class="nav-link active" data-toggle="tab" href="#import-local-tab" role="tab">Local</a>
						</li>
						<li class="nav-item">
							<a class="nav-link" data-toggle="tab" href="#import-upload-tab" role="tab">Upload</a>
						</li>
					</ul>
					<div class="tab-content">
						<div class="tab-pane show active" id="import-local-tab">
							<form v-on:submit.prevent="submitLocal">
								<div class="row mb-2">
									<label class="col-sm-2 col-form-label">Path</label>
									<div class="col-sm-10">
										<input class="form-control" type="text" v-model="path" />
										<small class="form-text text-muted">
											<template v-if="mode == 'add'">
												The path to a file or directory from which to import files.
												The path must exist on the local disk where SkyhookML is running.
											</template>
											<template v-if="mode == 'new'">
												The path to a SkyhookML-formatted archive (.zip or directory containing db.sqlite3 and files) on the local disk where SkyhookML is running.
											</template>
										</small>
									</div>
								</div>
								<div class="row mb-2">
									<div class="col-sm-2">Symlink</div>
									<div class="col-sm-10">
										<div class="form-check">
											<input class="form-check-input" type="checkbox" v-model="symlink">
											<label class="form-check-label">
												Symlink instead of copying when possible.
											</label>
										</div>
									</div>
								</div>
								<div class="row">
									<div class="col-sm-10">
										<button type="submit" class="btn btn-primary">Import</button>
									</div>
								</div>
							</form>
						</div>
						<div class="tab-pane" id="import-upload-tab">
							<form v-on:submit.prevent="submitUpload">
								<div class="row mb-2">
									<label class="col-sm-2 col-form-label">File</label>
									<div class="col-sm-10">
										<input class="form-control" type="file" @change="onFileChange" />
										<small class="form-text text-muted">
											<template v-if="mode == 'add'">
												<template v-if="dataset.DataType == 'video'">
													Video files (e.g., mp4) or a zip file that contains them.
												</template>
												<template v-else-if="dataset.DataType == 'image'">
													Image files (PNG or JPG) or a zip file that contains them.
												</template>
												<template v-else-if="dataset.DataType == 'detection' || dataset.DataType == 'int' || dataset.DataType == 'shape' || dataset.DataType == 'floats'">
													Data in SkyhookML JSON format (either .json file or zip file containing .json).
													To import data in other formats, use <router-link :to="'/ws/'+$route.params.ws+'/quickstart/import'">Quickstart/Import</router-link>.
												</template>
												<template v-else-if="dataset.DataType == 'file'">
													Either files or a zip file.
												</template>
												<template v-else>
													Data in a SkyhookML-supported format.
													To import data in other formats, use <router-link :to="'/ws/'+$route.params.ws+'/quickstart/import'">Quickstart/Import</router-link>.
												</template>
											</template>
											<template v-else-if="mode == 'new'">
												A SkyhookML-formatted dataset archive (.zip containing db.sqlite3 and files).
											</template>
										</small>
									</div>
								</div>
								<div class="row">
									<div class="col-sm-10">
										<button type="submit" class="btn btn-primary">Import</button>
									</div>
								</div>
							</form>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</span>
	`,
};
