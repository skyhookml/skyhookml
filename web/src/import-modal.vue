<template>
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
							<button class="nav-link active" data-bs-toggle="tab" data-bs-target="#import-local-tab" role="tab">Local</button>
						</li>
						<li class="nav-item">
							<button class="nav-link" data-bs-toggle="tab" data-bs-target="#import-upload-tab" role="tab">Upload</button>
						</li>
						<li class="nav-item">
							<button class="nav-link" data-bs-toggle="tab" data-bs-target="#import-url-tab" role="tab">URL</button>
						</li>
					</ul>
					<div class="tab-content">
						<div class="tab-pane show active" id="import-local-tab">
							<form v-on:submit.prevent="submitLocal">
								<div class="row mb-2">
									<label class="col-sm-2 col-form-label">Path</label>
									<div class="col-sm-10">
										<input class="form-control" type="text" v-model="path" required />
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
							<template v-if="percent === null">
								<form v-on:submit.prevent="submitUpload">
									<div class="row mb-2">
										<label class="col-sm-2 col-form-label">File</label>
										<div class="col-sm-10">
											<input class="form-control" type="file" @change="onFileChange" required />
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
							</template>
							<template v-else>
								<h2>Uploading...</h2>
								<div class="mt-2">
									<div class="progress">
										<div
											class="progress-bar"
											role="progressbar"
											v-bind:style="{width: percent+'%'}"
											:aria-valuenow="percent"
											aria-valuemin="0"
											aria-valuemax="100"
											>
											{{ percent }}%
										</div>
									</div>
								</div>
							</template>
						</div>
						<div class="tab-pane" id="import-url-tab">
							<form v-on:submit.prevent="submitURL">
								<div class="row mb-2">
									<label class="col-sm-2 col-form-label">URL</label>
									<div class="col-sm-10">
										<input class="form-control" type="text" v-model="url" required />
										<small class="form-text text-muted">
											<template v-if="mode == 'add'">
												The URL of a zip file from which to import files.
											</template>
											<template v-if="mode == 'new'">
												The URL of a SkyhookML-formatted dataset archive (.zip containing db.sqlite3 and files).
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
</template>

<script>
import utils from './utils.js';

export default {
	data: function() {
		return {
			path: '',
			symlink: false,
			file: null,
			percent: null,
			url: null,

			// either /datasets/[ID]/import or /import-dataset depending on mode
			importEndpoint: '',
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
			this.importEndpoint = '/datasets/'+this.dataset.ID+'/import';
		} else if(this.mode == 'new') {
			this.importEndpoint = '/import-dataset';
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
			utils.request(this, 'POST', this.importEndpoint+'?mode=local', params, (job) => {
				this.$router.push('/ws/'+this.$route.params.ws+'/jobs/'+job.ID);
			});
			$(this.$refs.modal).modal('hide');
		},
		submitUpload: function() {
			var data = new FormData();
			data.append('file', this.file);
			this.percent = 0;
			$.ajax({
				type: 'POST',
				url: this.importEndpoint+'?mode=upload',
				error: (req, status, errorMsg) => {
					this.percent = null;
					$(this.$refs.modal).modal('hide');
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
						this.percent = Math.min(parseInt(e.loaded * 100 / e.total), 99);
					});
					return xhr;
				},
				success: (job) => {
					this.percent = null;
					$(this.$refs.modal).modal('hide');
					this.$router.push('/ws/'+this.$route.params.ws+'/jobs/'+job.ID);
				},
			});
		},
		submitURL: function() {
			let params = {
				mode: 'url',
				url: this.url,
			};
			utils.request(this, 'POST', this.importEndpoint+'?mode=url', params, (job) => {
				this.$router.push('/ws/'+this.$route.params.ws+'/jobs/'+job.ID);
			});
			$(this.$refs.modal).modal('hide');
		},
	},
};
</script>