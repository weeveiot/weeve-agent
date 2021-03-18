Following docker-compose [file specification](https://github.com/compose-spec/compose-spec/blob/master/spec.md) version for docker compose 1.27.0.

The data service compose file.

The data service compose file works with containers which have an ENTRYPOINT defined.

All weeve attributes begin with `w_`.

The following keys are part of the spec:
- **w_data-service-name** This is the name of the data service.
- **w_data-service-id** A unique identifier for the data service compose file.
- **services** Each image will be instantiated as a service. Each service will named as an object.
    - **services/[name]** Each service is named at the object level.
        - **services/[name]/networks** The docker network is named here. Only the bridge network driver is supported.
        - **services/[name]/command/[array]** All strings are appended to the Dockerfile ENTRYPOINT, provided entrypoint is defined in exec form in the Dockerfile.
        Command overrides the the default command declared by the container image (i.e. by Dockerfile's CMD). command: bundle exec thin -p 3000 The command can also be a list, in a manner similar to Dockerfile.
        - **services/[name]/image**  Image MUST follow the Open Container Specification addressable image format, as [<registry>/][<project>/]<image>[:<tag>|@<digest>].
        - **services/[name]/entrypoint** UNSUPPORTED To override the default entrypoint, use entrypoint option. To pass the arguments use command.
        - **services/[name]/build** UNSUPPORTED Build is not used on the edge devices, an image will always be pulled from the image registry
- **networks**
- **volumes**
- **configs**
- **secrets**

The following docker compose keys will be supported in a future version


