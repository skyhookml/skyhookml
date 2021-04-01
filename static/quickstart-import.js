import utils from './utils.js';
import JobConsoleProgress from './job-consoleprogress.js';

export default {
	components: {
		'job-consoleprogress': JobConsoleProgress,
	},
	data: function() {
		return {
			// which phase are we on?
			// (1) 'type': pick import type like 'unlabeled-video' or 'object-detection'
			// (2) 'format': if not unlabeled, pick data format like 'yolo' or 'catfolder'
			// (3) 'upload': select zip file to upload
			// (3.5) 'uploading'
			// (4) 'importing': unzipping and copying files
			// (5) 'converting': running convert op
			// (6) 'done'
			// also 'error'
			phase: 'type',

			// import data type
			importType: '',

			// import format
			format: '',

			// the job we're waiting for, if phase is 'importing' or 'converting'
			job: null,

			// if phase is error
			errorMsg: '',

			name: '',
			file: null,
			percent: null,
		};
	},
	methods: {
		selectImportType: function(t) {
			this.importType = t;
			if(this.importType == 'unlabeled-video' || this.importType == 'unlabeled-image') {
				this.phase = 'upload';
			} else {
				this.phase = 'format';
			}
		},
		selectFormat: function(format) {
			this.format = format;
			this.phase = 'upload';
		},
		onFileChange: function(event) {
			this.file = event.target.files[0];
		},
		submitUpload: function() {
			let handle = async () => {
				this.phase = 'uploading';

				// determine what steps are needed
				// unlabeled: can just add the files directly to dataset
				// otherwise: need to upload to a file dataset and then convert to Skyhook format
				let uploadType = null;
				let dtypes = null;
				let convertOp = null;
				if(this.importType == 'unlabeled-video') {
					uploadType = 'video';
				} else if(this.importType == 'unlabeled-image') {
					uploadType = 'image';
				} else {
					uploadType = 'file';
					dtypes = {};
					if(this.format == 'yolo') {
						dtypes.images = 'image';
						dtypes.detections = 'detection';
						convertOp = 'from_yolo';
					} else if(this.format == 'coco') {
						dtypes.images = 'image';
						dtypes.detections = 'detection';
						convertOp = 'from_coco';
					} else if(this.format == 'catfolder') {
						dtypes.images = 'image';
						dtypes.labels = 'int';
						convertOp = 'from_catfolder';
					}
				}
				console.log('[import] determined steps', 'uploadType=', uploadType, 'dtypes=', dtypes, 'convertOp=', convertOp);

				// prepare cleanup system to delete things that we make in case there ends up being an error
				let cleanupFuncs = [];
				let cleanup = () => {
					for(let f of cleanupFuncs) {
						f();
					}
				};
				let setError = (errorMsg, e) => {
					console.log(errorMsg, e);
					this.phase = 'error';
					this.errorMsg = errorMsg;
					cleanup();
				}

				// create a new dataset
				let params = {data_type: uploadType};
				if(uploadType == 'file') {
					params.name = this.name + '-file';
				} else {
					params.name = this.name;
				}
				let dataset;
				try {
					dataset = await utils.request(this, 'POST', '/datasets', params);
				} catch(e) {
					setError('Error creating dataset for upload: ' + e.responseText, e);
					return;
				}
				console.log('[import] created dataset', dataset.ID);
				cleanupFuncs.push(() => {
					utils.request(this, 'DELETE', '/datasets/'+dataset.ID);
				});

				// upload the archive
				var data = new FormData();
				data.append('file', this.file);
				this.percent = null;
				let importJob;
				try {
					importJob = await $.ajax({
						type: 'POST',
						url: '/datasets/'+dataset.ID+'/import?mode=upload',
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
					});
				} catch(e) {
					setError('Error uploading: ' + e.responseText, e);
					return;
				}

				// wait for import to complete
				this.job = importJob;
				this.phase = 'importing';
				console.log('[import] waiting for import job');
				try {
					await utils.waitForJob(this.job.ID);
				} catch(e) {
					setError('Import error: ' + e.Error);
					return;
				}
				console.log('[import] import job completed');

				// we're done if we don't need to convert
				// otherwise we need to do that
				if(!convertOp) {
					this.phase = 'done';
					cleanup();
					return;
				}

				// create output datasets
				let outDatasets = {};
				cleanupFuncs.push(() => {
					// add cleanup func here since there may be partial failure when creating output datasets
					for(let ds of Object.values(outDatasets)) {
						if(!ds) {
							continue;
						}
						utils.request(this, 'DELETE', '/datasets/'+ds.ID);
					}
				});
				let promises = [];
				let successCount = 0;
				for(let [name, dtype] of Object.entries(dtypes)) {
					outDatasets[name] = null;
					let params = {
						name: this.name+'-'+name,
						data_type: dtype,
					};
					let promise = utils.request(this, 'POST', '/datasets', params, (ds) => {
						outDatasets[name] = ds;
						successCount++;
					});
					promises.push(promise);
				}
				try {
					await Promise.all(promises);
				} catch(e) {
					setError('Error creating output datasets: ' + e.responseText, e);
					return;
				}
				console.log('created output datasets', outDatasets);

				// create convert job
				// we do this by creating an anonymous runnable
				let runnable = {
					Name: 'quickstart-import-convert',
					Op: convertOp,
					Params: '',
					InputDatasets: {'input': [dataset]},
					OutputDatasets: outDatasets,
				};
				let convertJob;
				try {
					convertJob = await utils.request(this, 'POST', '/runnable', JSON.stringify(runnable));
				} catch(e) {
					setError('Error starting convert job: ' + e.responseText, e);
					return;
				}


				// wait for convert to complete
				this.job = convertJob;
				this.phase = 'converting';
				console.log('[import] waiting for convert job');
				try {
					await utils.waitForJob(this.job.ID);
				} catch(e) {
					setError('Conversion error: ' + e.Error);
					return;
				}
				console.log('[import] convert job completed');
				this.phase = 'done';
			};
			handle();
		},
	},
	template: `
<div class="flex-container">
	<template v-if="phase == 'type'">
		<h3>Quick Import</h3>
		<p>Select the type of data to import:</p>
		<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectImportType('unlabeled-image')">
			<div class="card-body">
				<h5 class="card-title">Unlabeled Images</h5>
			</div>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectImportType('unlabeled-video')">
			<div class="card-body">
				<h5 class="card-title">Unlabeled Video</h5>
			</div>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectImportType('object-detection')">
			<div class="card-body">
				<h5 class="card-title">Object Detection Labels</h5>
				<p class="card-text">Object detection annotations, optionally with corresponding image or video.</p>
			</div>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectImportType('image-classification')">
			<div class="card-body">
				<h5 class="card-title">Image Classification Labels</h5>
				<p class="card-text">Image classification annotations, optionally with corresponding image or video.</p>
			</div>
		</div>
	</template>
	<template v-else-if="phase == 'format'">
		<h3>Quick Import</h3>
		<p>What format is your data in?</p>
		<template v-if="importType == 'object-detection'">
			<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectFormat('yolo')">
				<div class="card-body">
					<h5 class="card-title">YOLO</h5>
					<p class="card-text">Images and YOLO-formatted .txt files with matching filenames.</p>
					<p class="card-text"><small class="text-muted">Each line in the .txt files should be formatted as "[category id] [cx] [cy] [w] [h]", with coordinates normalized to be between 0-1.</small></p>
				</div>
			</div>
			<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectFormat('coco')">
				<div class="card-body">
					<h5 class="card-title">COCO-JSON</h5>
					<p class="card-text">Images and a JSON file in COCO format specifying the instances in each image.</p>
				</div>
			</div>
		</template>
		<template v-if="importType == 'image-classification'">
			<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectFormat('catfolder')">
				<div class="card-body">
					<h5 class="card-title">Category-Folders</h5>
					<p class="card-text">An archive containing folders that are named corresponding to object categories. An image should be appear in the folder matching its category.</p>
					<p class="card-text"><small class="text-muted">Example: dogs/1.jpg, dogs/2.jpg, cats/3.jpg.</small></p>
				</div>
			</div>
		</template>
	</template>
	<template v-else-if="phase == 'upload'">
		<h3>Quick Import</h3>
		<p>Upload a zip archive:</p>
		<form v-on:submit.prevent="submitUpload" class="small-container">
			<div class="row mb-2">
				<label class="col-sm-4 col-form-label">File</label>
				<div class="col-sm-8">
					<input class="form-control" type="file" @change="onFileChange" />
					<small class="form-text text-muted">
						<template v-if="importType == 'unlabeled-video'">
							Upload a zip archive containing video files (e.g. mp4).
						</template>
						<template v-else-if="importType == 'unlabeled-image'">
							Upload a zip archive containing image files (JPG or PNG).
						</template>
						<template v-else-if="format == 'yolo'">
							Upload a zip archive containing images (JPG or PNG) and corresponding YOLO-formatted .txt files.
							The archive can also optionally contain an obj.names file that maps category IDs to names.
						</template>
						<template v-else-if="format == 'coco'">
							Upload a zip archive containing images (JPG or PNG) and a COCO-formatted annotation JSON file.
							The archive can contain subfolders, and the JSON file can be in a different folder than the images, but there must be exactly one JSON file.
						</template>
					</small>
				</div>
			</div>
			<div class="row mb-2">
				<label class="col-sm-4 col-form-label">Dataset Name</label>
				<div class="col-sm-8">
					<input class="form-control" type="text" v-model="name" />
					<small class="form-text text-muted">
						A label for this dataset.
					</small>
				</div>
			</div>
			<div class="row">
				<div class="col-sm-12">
					<button type="submit" class="btn btn-primary">Import</button>
				</div>
			</div>
		</form>
	</template>
	<template v-else-if="phase == 'uploading'">
		<h3>Uploading...</h3>
	</template>
	<template v-else-if="phase == 'importing' || phase == 'converting'">
		<h3>
			<template v-if="phase == 'importing'">
				Importing...
			</template>
			<template v-else-if="phase == 'converting'">
				Converting...
			</template>
		</h3>
		<div class="flex-content">
			<job-consoleprogress :jobID="job.ID"></job-consoleprogress>
		</div>
	</template>
	<template v-else-if="phase == 'done'">
		<div class="small-container">
			<p>The import has completed successfully.</p>
			<router-link class="btn btn-primary" :to="'/ws/'+$route.params.ws+'/datasets'">Go to Datasets</router-link>
		</div>
	</template>
	<template v-else-if="phase == 'error'">
		<div class="small-container">
			<h3>Error Importing Data</h3>
			<p>{{ errorMsg }}</p>
		</div>
	</template>
</div>
	`,
};
