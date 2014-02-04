fae
===

         ____      __      ____ 
        ( ___)    /__\    ( ___)
         )__)    /(__)\    )__) 
        (__)    (__)(__)  (____)
                               
Fun App Engine

It's middleware RPC engine.

### Why?

*   Seperation of concerns
*   Reuse common code as service and transparently reuse infrastructure
*   Centralized best practice
*   Centralized monitoring, auditting and profiling
*   lessen tcp 3/4 way handshake overhead(conn pooling)
*   Scale
*   Polyglot development

### Features

*   local LRU cache shared among processes
*   memcache servant
*   mongodb servant with transaction support
*   distributed logger servant

### Requirement

    git clone https://github.com/apache/thrift.git
    cd thrift
    ./bootstrap.sh
    ./configure --prefix=/opt/app/thrift
    make
    make -k check
    sh test/test.sh
    make install

### Architecture


        +----------------+  +----------------+  +----------------+
        | php-fpm worker |  | php-fpm worker |  | php-fpm worker |
        +----------------+  +----------------+  +----------------+
            |                       |                       |
             -----------------------------------------------
                                    |                        
                                    | tcp/unix socket
                                    |                        
                            +---------------+
                            |  faed daemon  |
                            +---------------+
                            |  local cache  | 
                            +---------------+
                                    |                        
                                    | tcp long connection pool(keepalive)
                                    |                        
             -----------------------------------------------
            |                       |                       | hierarchy
        +----------------+  +----------------+  +----------------+
        | mongodb servers|  |memcache servers|  |   faed daemon  |
        +----------------+  +----------------+  +----------------+

