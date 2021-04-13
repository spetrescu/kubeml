import subprocess
from .experiment import *
from hashlib import sha256
from functools import wraps


def retry(exceptions, total_tries=4, initial_wait=0.5, backoff_factor=2):
    """
    calling the decorated function applying an exponential backoff.
    Args:
        exceptions: Exeption(s) that trigger a retry, can be a tuple
        total_tries: Total tries
        initial_wait: Time to first retry
        backoff_factor: Backoff multiplier (e.g. value of 2 will double the delay each retry).
        logger: logger to be used, if none specified print
    """
    def retry_decorator(f):
        @wraps(f)
        def func_with_retries(*args, **kwargs):
            _tries, _delay = total_tries + 1, initial_wait
            while _tries > 1:
                try:
                    print(f'{total_tries + 2 - _tries}. try:')
                    return f(*args, **kwargs)
                except exceptions as e:
                    _tries -= 1
                    print_args = args if args else 'no args'
                    if _tries == 1:
                        msg = str(f'Function: {f.__name__}\n'
                                  f'Failed despite best efforts after {total_tries} tries.\n'
                                  f'args: {print_args}, kwargs: {kwargs}')
                        print(msg)
                        raise
                    msg = str(f'Function: {f.__name__}\n'
                              f'Exception: {e}\n'
                              f'Retrying in {_delay} seconds!, args: {print_args}, kwargs: {kwargs}\n')
                    print(msg)
                    time.sleep(_delay)
                    _delay *= backoff_factor
        return func_with_retries
    return retry_decorator


def get_title(req) -> str:
    return f'{req.function_name}-batch{req.batch_size}-k{req.options.k}-parallel{req.options.default_parallelism}-TTA{req.options.goal_accuracy}'


def get_hash(title: str) -> str:
    """Given the experiment title return a hash so experimets with the same params
    can be identified between replications"""
    return sha256(title.encode('utf-8')).hexdigest()[:16]


def check_stderr(res: subprocess.CompletedProcess):
    """Checks whether the executed command returned an error and exists if that is the case"""
    if len(res.stderr) == 0:
        return
    print("error running command", res.args, res.stderr.decode())
    raise Exception


def create_function(name: str, file: str):
    """Creates a function in kubeml"""

    command = f"kubeml fn create --name {name} --code {file}"
    print("Creating function", name)

    res = subprocess.run(command.split(), stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    check_stderr(res)

    print("function created")
